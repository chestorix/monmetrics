package main

import "flag"

var (
	flagRunAddr        string
	flagReportInterval int
	flagPollInterval   int
	flagKey            string
	flagRateLimit      int
)

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "interval to report metrics (seconds)")
	flag.IntVar(&flagPollInterval, "p", 2, "interval to poll metrics (seconds)")
	flag.StringVar(&flagKey, "k", "", "secret key")
	flag.IntVar(&flagRateLimit, "l", 1, "rate limit for outgoing requests")
	flag.Parse()
}
