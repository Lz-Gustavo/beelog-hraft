package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
)

var (
	svrID            string
	svrPort          string
	raftAddr         string
	joinAddr         string
	joinHandlerAddr  string
	recovHandlerAddr string
	cpuprofile       *string
	memprofile       *string
	logfolder        *string
)

func init() {
	parseIPsFromArgsConfig()
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to a file")
	memprofile = flag.String("memprofile", "", "write memory profile to a file")
	logfolder = flag.String("logfolder", "", "log received commands to a file at specified destination folder")
}

func main() {
	parseAndDebugConfig()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize the Key-value store
	kvs := NewStore(ctx, true)
	listener, err := net.Listen("tcp", svrPort)
	if err != nil {
		log.Fatalf("failed to start connection: %s", err.Error())
	}

	// Start the Raft cluster
	if err := kvs.StartRaft(joinAddr == "", svrID, raftAddr); err != nil {
		log.Fatalf("failed to start raft cluster: %s", err.Error())
	}

	// Initialize the server
	server := NewServer(ctx, kvs)

	// Send a join request, if any
	if joinAddr != "" {
		if err = sendJoinRequest(); err != nil {
			log.Fatalf("failed to send join request to node at %s: %s", joinAddr, err.Error())
		}
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("accept failed: %s", err.Error())
			}
			server.joins <- conn
		}
	}()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
	cancel()
	server.Exit()
}

func sendJoinRequest() error {
	joinConn, err := net.Dial("tcp", joinAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to leader node at %s: %s", joinAddr, err.Error())
	}

	_, err = fmt.Fprint(joinConn, svrID+"-"+raftAddr+"-"+"true"+"\n")
	if err != nil {
		return fmt.Errorf("failed to send join request to node at %s: %s", joinAddr, err.Error())
	}

	if err = joinConn.Close(); err != nil {
		return err
	}
	return nil
}

func parseIPsFromArgsConfig() {
	flag.StringVar(&svrID, "id", "", "Set server unique ID")
	flag.StringVar(&svrPort, "port", ":11000", "Set the server bind address")
	flag.StringVar(&raftAddr, "raft", ":12000", "Set RAFT consensus bind address")
	flag.StringVar(&joinAddr, "join", "", "Set join address, if any")
	flag.StringVar(&joinHandlerAddr, "hjoin", "", "Set port id to receive join requests on the raft cluster")
	flag.StringVar(&recovHandlerAddr, "hrecov", "", "Set port id to receive state transfer requests from the application log")
}

func parseAndDebugConfig() {
	flag.Parse()
	if svrID == "" {
		log.Fatalln("Must set a server ID, run with: ./server -id 'svrID'")
	}

	fmt.Println(
		"=========================",
		"\n=== Running with config:",
		"\nID:    ", svrID,
		"\napp:   ", svrPort,
		"\nraft:  ", raftAddr,
		"\njoin:  ", joinAddr,
		"\nhjoin: ", joinHandlerAddr,
		"\nhrecov:", recovHandlerAddr,
		"\n=========================",
	)
}
