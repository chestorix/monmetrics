package main

import "flag"

var (
	flagRunAddr         string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
)

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagStoreInterval, "i", 300, "interval in seconds to save metrics to disk (0 for synchronous)")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file path to save/load metrics")
	flag.BoolVar(&flagRestore, "r", true, "whether to restore metrics from file on startup")
	flag.Parse()
}
