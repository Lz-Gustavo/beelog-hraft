package main

import (
	"context"
	"testing"
	"time"

	"github.com/Lz-Gustavo/beelog/pb"

	"github.com/golang/protobuf/proto"
)

func TestCreate(t *testing.T) {
	s := NewStore(context.TODO(), true)
	if err := s.StartRaft(true, "node0", ":12000"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}

	// block until shutdown
	if err := s.raft.Shutdown().Error(); err != nil {
		t.Fatalf("failed to shutdown raft")
	}
}

func TestOperations(t *testing.T) {
	s := NewStore(context.TODO(), true)
	if err := s.StartRaft(true, "node0", ":12000"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)

	cmd := &pb.Command{
		Op:    pb.Command_SET,
		Key:   "foo",
		Value: "bar",
	}
	bytes, _ := proto.Marshal(cmd)
	if err := s.Propose(bytes, nil, ""); err != nil {
		t.Fatalf("failed to set key: %s", err.Error())
	}

	// Wait for committed log entry to be applied.
	time.Sleep(500 * time.Millisecond)
	value := s.testGet("foo")
	if value != "bar" {
		t.Fatalf("key has wrong value: %s", value)
	}
}
