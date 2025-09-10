package main

import "flag"

var (
	flagRunAddr         string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
	flagConnDB          string
	flagKey             string
	flagCryptoKey       string
	flagConfigFile      string
)

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagStoreInterval, "i", 10, "interval in seconds to save metrics to disk (0 for synchronous)")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file path to save/load metrics")
	flag.BoolVar(&flagRestore, "r", true, "whether to restore metrics from file on startup")
	flag.StringVar(&flagConnDB, "d", "", "host=<host> user=<user> password=<password> dbname=<dbname> sslmode=<disable/enable>")
	flag.StringVar(&flagKey, "k", "", "secret key")
	flag.StringVar(&flagCryptoKey, "crypto-key", "", "path to file private key")
	flag.StringVar(&flagConfigFile, "c", "", "path to config file")
	flag.StringVar(&flagConfigFile, "config", "", "path to config file")
	flag.Parse()
}
