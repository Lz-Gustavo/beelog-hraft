package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	bl "github.com/Lz-Gustavo/beelog"
	"github.com/Lz-Gustavo/beelog/pb"
)

var (
	initValue = []byte(strings.Repeat("!", initValueSize))
)

// MockState ...
type MockState struct {
	state map[string][]byte
}

// NewMockState ...
func NewMockState() *MockState {
	m := &MockState{
		state: make(map[string][]byte, numInitKeys),
	}

	if preInitialize {
		for i := 0; i < numInitKeys; i++ {
			m.state[strconv.Itoa(i)] = initValue
		}
	}
	return m
}

// InstallRecovState ...
func (m *MockState) InstallRecovState(newState []byte) (uint64, error) {
	rd := bytes.NewReader(newState)
	return m.InstallRecovStateFromReader(rd)
}

// InstallRecovStateFromReader ...
func (m *MockState) InstallRecovStateFromReader(rd io.Reader) (uint64, error) {
	cmds, err := bl.UnmarshalLogFromReader(rd)
	if err != nil {
		return 0, err
	}

	m.applyLog(cmds)
	return uint64(len(cmds)), nil
}

// InstallRecovStateForMultipleLogs ...
func (m *MockState) InstallRecovStateForMultipleLogs(newState []byte) (uint64, error) {
	rd := bytes.NewReader(newState)
	return m.InstallRecovStateForMultipleLogsFromReader(rd)
}

// InstallRecovStateForMultipleLogsFromReader ...
func (m *MockState) InstallRecovStateForMultipleLogsFromReader(rd io.Reader) (uint64, error) {
	var nLogs int

	// read num of retrieved logs, only on multiple logs config
	_, err := fmt.Fscanf(rd, "%d\n", &nLogs)
	if err != nil {
		return 0, err
	}

	// TODO: currently considers an ordered sequence of logs being informed by replicas,
	// which increases recovery time and thus, compromises availability. Modify solution
	// when unordered logs can be returned by the conctable.RecovEntireLog()
	//
	// NOTE: not a priority for now
	var nCmds uint64
	for i := 0; i < nLogs; i++ {
		cmds, err := bl.UnmarshalLogFromReader(rd)
		if err != nil {
			return 0, err
		}

		nCmds += uint64(len(cmds))
		m.applyLog(cmds)
	}
	return nCmds, nil
}

// applyLog executes received commands on mock state.
func (m *MockState) applyLog(log []pb.Command) {
	for _, cmd := range log {
		switch cmd.Op {
		case pb.Command_SET:
			m.state[cmd.Key] = []byte(cmd.Value)

		default:
			break
		}
	}
}

// applyLogCountingDiffKeys executes received commands on mock state, returning the
// number of different keys identified.
func (m *MockState) applyLogCountingDiffKeys(log []pb.Command) int {
	diff := 0
	for _, cmd := range log {
		if _, ok := m.state[cmd.Key]; !ok {
			diff++
		}

		switch cmd.Op {
		case pb.Command_SET:
			m.state[cmd.Key] = []byte(cmd.Value)

		case pb.Command_GET:
			// insert an empty value to count READ operations on unique keys
			m.state[cmd.Key] = []byte{}

		default:
			break
		}
	}
	return diff
}
