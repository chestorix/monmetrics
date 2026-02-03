package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
)

var (
	ErrInvalidKey = errors.New("invalid key")
)

// LoadPublicKey загружает публичный ключ из файла

func LoadPublicKey(filename string) (*rsa.PublicKey, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrInvalidKey
	}

	var pub interface{}

	if block.Type == "PUBLIC KEY" {
		pub, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	} else if block.Type == "RSA PUBLIC KEY" {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrInvalidKey
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, ErrInvalidKey
	}

	return publicKey, nil
}

// LoadPrivateKey загружает приватный ключ из файла
func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrInvalidKey
	}

	var priv interface{}

	if block.Type == "RSA PRIVATE KEY" {
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	} else if block.Type == "PRIVATE KEY" {
		priv, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrInvalidKey
	}

	privateKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrInvalidKey
	}

	return privateKey, nil
}

func EncryptData(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return data, nil
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encryptedData := gcm.Seal(nonce, nonce, data, nil)

	encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key with RSA: %w", err)
	}

	log.Printf("Encrypted AES key size: %d bytes", len(encryptedKey))
	log.Printf("Encrypted data size: %d bytes", len(encryptedData))

	keySize := len(encryptedKey)
	result := make([]byte, 4+keySize+len(encryptedData))

	binary.BigEndian.PutUint32(result[0:4], uint32(keySize))
	log.Printf("Writing key size to header: %d bytes", keySize)

	copy(result[4:4+keySize], encryptedKey)
	copy(result[4+keySize:], encryptedData)

	log.Printf("Total encrypted size: %d bytes", len(result))
	return result, nil
}

// DecryptData расшифровывает данные приватным ключом
func DecryptData(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return data, nil
	}

	log.Printf("DecryptData: input size %d bytes", len(data))

	if len(data) < 4 {
		return nil, fmt.Errorf("invalid encrypted data: too short for key size header")
	}

	keySize := int(binary.BigEndian.Uint32(data[0:4]))
	log.Printf("Key size from header: %d bytes", keySize)

	if len(data) < 4+keySize {
		return nil, fmt.Errorf("invalid encrypted data: too short, need %d bytes for key, have %d",
			4+keySize, len(data))
	}

	encryptedKey := data[4 : 4+keySize]
	encryptedData := data[4+keySize:]

	log.Printf("Encrypted key: %d bytes, encrypted data: %d bytes",
		len(encryptedKey), len(encryptedData))

	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}
	log.Printf("AES key decrypted: %d bytes", len(aesKey))

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	log.Printf("GCM nonce size: %d bytes", nonceSize)

	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("invalid encrypted data: missing nonce")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	log.Printf("Nonce: %d bytes, ciphertext: %d bytes", len(nonce), len(ciphertext))

	decryptedData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	log.Printf("Successfully decrypted to %d bytes", len(decryptedData))
	return decryptedData, nil
}
