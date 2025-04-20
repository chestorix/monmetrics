package main

import "flag"

var (
	flagRunAddr        string
	flagReportInterval int
	flagPollInterval   int
)

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "http://localhost:8080", "address and port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "interval to report metrics (seconds)")
	flag.IntVar(&flagPollInterval, "p", 2, "interval to poll metrics (seconds)")
	flag.Parse()
}
