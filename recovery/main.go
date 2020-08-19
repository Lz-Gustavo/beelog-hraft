package main

import (
	"flag"
	"log"
)

const (
	preInitialize = false
	numInitKeys   = 1000000
	initValueSize = 128
)

var (
	// switches execution between the log verifier and state requester.
	checkMode bool

	// used to initialize a state transfer protocol to the application log after a
	// specified number of seconds.
	sleepDuration int

	recovAddr             string
	firstIndex, lastIndex string
	multipleLogs          bool
)

func init() {
	flag.BoolVar(&checkMode, "check", false, "switches execution between the log verifier and state requester, defaults to the latter")
	flag.IntVar(&sleepDuration, "sleep", 0, "set a countdown in seconds for a state request, defaults to none (0s)")
	flag.StringVar(&recovAddr, "recov", ":14000", "set an address to request state, defaults to localhost:14000")
	flag.StringVar(&firstIndex, "p", "", "set the first index of requested state")
	flag.StringVar(&lastIndex, "n", "", "set the last index of requested state")
	flag.BoolVar(&multipleLogs, "mult", false, "inform wheter multiple logs will be returned")
}

func main() {
	flag.Parse()
	if checkMode {
		err := checkLocalLogs()
		if err != nil {
			log.Fatalln("failed logs verification with err: '", err.Error(), "'")
		}
	} else {
		err := requestLogs()
		if err != nil {
			log.Fatalln("failed log requester with err: '", err.Error(), "'")
		}
	}
}
