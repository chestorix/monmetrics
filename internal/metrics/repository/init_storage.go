// Package repository - реализация хранилища.
package repository

import (
	"fmt"
	"github.com/chestorix/monmetrics/internal/domain/interfaces"
	"github.com/chestorix/monmetrics/internal/utils"
	"time"
)

type InitStorage struct {
	retryDelays []time.Duration
}

func NewInitStorage() *InitStorage {
	return &InitStorage{
		retryDelays: []time.Duration{time.Second, time.Second * 3, time.Second * 5},
	}
}

func (i *InitStorage) CreateStorage(dbDSN, filePath string) (interfaces.Repository, error) {
	var storage interfaces.Repository
	var err error

	if dbDSN != "" {
		fmt.Println("Print dbDSN", dbDSN)
		err = utils.Retry(3, i.retryDelays, func() error {
			storage, err = NewPostgresStorage(dbDSN)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL storage: %w", err)
		}
	} else if filePath != "" {
		storage = NewMemStorage(filePath)
	} else {
		storage = NewMemStorage("")
	}

	return storage, nil
}
