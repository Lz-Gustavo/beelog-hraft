package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

var (
	numApps          int
	logIDs           []string
	raftAddrs        []string
	joinAddrs        []string
	recovHandlerAddr string
	logfolder        *string
)

func init() {
	err := parseIPsFromArgsConfig()
	if err != nil {
		log.Fatalln("could not parse cmdli args, err:", err.Error())
	}
	debugLoggerState()
}

func main() {
	loggerInstances := make([]*Logger, numApps)
	for i := 0; i < numApps; i++ {
		go func(j int) {

			loggerInstances[j] = NewLogger(logIDs[j])
			if err := loggerInstances[j].StartRaft(logIDs[j], raftAddrs[j]); err != nil {
				log.Fatalf("failed to start raft cluster: %s", err.Error())
			}
			if err := sendJoinRequest(logIDs[j], raftAddrs[j], joinAddrs[j]); err != nil {
				log.Fatalf("failed to send join request to node at %s: %s", joinAddrs[j], err.Error())
			}
		}(i)
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate

	for _, l := range loggerInstances {
		l.cancel()
		l.LogFile.Close()
	}
}

func sendJoinRequest(logID, raftAddr, joinAddr string) error {
	joinConn, err := net.Dial("tcp", joinAddr)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(joinConn, logID+"-"+raftAddr+"-"+"false"+"\n")
	if err != nil {
		return err
	}
	err = joinConn.Close()
	if err != nil {
		return err
	}
	return nil
}

func countDiffStrInSlice(elements []string) int {
	foundMarker := make(map[string]bool, len(elements))
	numDiff := 0

	for _, str := range elements {
		if !foundMarker[str] {
			foundMarker[str] = true
			numDiff++
		}
	}
	return numDiff
}

func parseIPsFromArgsConfig() error {
	var logs, raft, joins string
	flag.StringVar(&logs, "id", "", "Set the logger unique ID")
	flag.StringVar(&raft, "raft", ":12000", "Set RAFT consensus bind address")
	flag.StringVar(&joins, "join", ":13000", "Set join address to an already configured raft node")
	flag.StringVar(&recovHandlerAddr, "hrecov", "", "Set port id to receive state transfer requests from the application log")
	flag.Parse()

	if logs == "" {
		return errors.New("must set a logger ID, run with: ./logger -id 'logID'")
	}

	logIDs = strings.Split(logs, ",")
	numDiffIds := countDiffStrInSlice(logIDs)

	raftAddrs = strings.Split(raft, ",")
	numDiffRaft := countDiffStrInSlice(raftAddrs)

	joinAddrs = strings.Split(joins, ",")
	numDiffServices := countDiffStrInSlice(joinAddrs)

	inconsistentQtd := numDiffServices != numDiffIds || numDiffIds != numDiffRaft || numDiffRaft != numDiffServices
	if inconsistentQtd {
		return errors.New("must run with the same number of unique IDs, raft and join addrs: ./logger -id 'X,Y' -raft 'A,B' -join 'W,Z'")
	}
	numApps = numDiffIds
	return nil
}

func debugLoggerState() {
	for i := 0; i < numApps; i++ {
		fmt.Println(
			"==========",
			"\nApplication #:", i,
			"\nloggerID:", logIDs[i],
			"\nraft:", raftAddrs[i],
			"\nappIP:", joinAddrs[i],
			"\n==========",
		)
	}
}
