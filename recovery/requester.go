package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"
)

func requestLogs() error {
	validIP := recovAddr != ""
	if !validIP {
		return fmt.Errorf("must set a valid IP address to request state, run with: ./recovery -recov 'ipAddress'")
	}
	if multipleLogs {
		fmt.Println("mult=TRUE, Expecting multiple logs...")
	} else {
		fmt.Println("mult=FALSE, Expecting a single log...")
	}

	if err := validInterval(firstIndex, lastIndex); err != nil {
		return fmt.Errorf("%s, must set a valid interval, run with: ./recovery -p 'num' -n 'num'", err.Error())
	}
	recovReplica := NewMockState()

	// Wait for the application to log a sequence of commands
	time.Sleep(time.Duration(sleepDuration) * time.Second)

	fmt.Printf("Asking for interval: [%s, %s]\n", firstIndex, lastIndex)
	state, dur := AskForStateTransfer(firstIndex, lastIndex)
	num, installDur := MeasureStateInstallation(recovReplica, state)

	fmt.Println(
		"=========================",
		"\nTransfer time (ns):", dur,
		"\nInstall time (ns): ", installDur,
		"\nNum of commands:   ", num,
		"\nState size (bytes):", len(state),
	)
	return nil
}

// AskForStateTransfer ...
func AskForStateTransfer(p, n string) ([]byte, uint64) {
	start := time.Now()
	recvState, err := sendStateRequest(p, n)
	if err != nil {
		log.Fatalf("failed to receive a new state from node '%s', error: %s", recovAddr, err.Error())
	}
	finish := uint64(time.Since(start) / time.Nanosecond)
	return recvState, finish
}

// MeasureStateInstallation ...
func MeasureStateInstallation(replica *MockState, recvState []byte) (nCmds, dur uint64) {
	var cmds uint64
	var err error

	start := time.Now()
	if multipleLogs {
		cmds, err = replica.InstallRecovStateForMultipleLogs(recvState)
		if err != nil {
			log.Fatalf("failed to install the received state: %s", err.Error())
		}

	} else {
		cmds, err = replica.InstallRecovState(recvState)
		if err != nil {
			log.Fatalf("failed to install the received state: %s", err.Error())
		}
	}
	finish := uint64(time.Since(start) / time.Nanosecond)
	return cmds, finish
}

func sendStateRequest(first, last string) ([]byte, error) {
	stateConn, err := net.Dial("tcp", recovAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node at '%s', error: %s", recovAddr, err.Error())
	}

	reqMsg := stateConn.LocalAddr().String() + "-" + first + "-" + last + "\n"
	_, err = fmt.Fprint(stateConn, reqMsg)
	if err != nil {
		return nil, fmt.Errorf("failed sending state request to node at '%s', error: %s", recovAddr, err.Error())
	}

	var recv []byte
	recv, err = ioutil.ReadAll(stateConn)
	if err != nil {
		return nil, fmt.Errorf("could not read state response: %s", err.Error())
	}

	if err = stateConn.Close(); err != nil {
		return nil, err
	}
	return recv, nil
}

func validInterval(first, last string) error {
	if first == "" || last == "" {
		return fmt.Errorf("empty string informed: '%s' or '%s'", first, last)
	}

	p, err := strconv.ParseUint(first, 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse '%s' as uint64", first)
	}

	n, err := strconv.ParseUint(last, 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse '%s' as uint64", last)
	}

	if p > n {
		return fmt.Errorf("invalid interval [%s, %s] informed", first, last)
	}
	return nil
}
