package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	disktradLogs = "/tmp/logfile*.log"
	beelogLogs   = "/tmp/beelog*.log"
)

func checkLocalLogs() error {
	fmt.Println(
		"=========================",
		"\nrunning log verifier...",
		"\n=========================",
	)

	dlogs, err := filepath.Glob(disktradLogs)
	if err != nil {
		return err
	}
	dlogs = rmvRepetitiveLogs(dlogs)

	blogs, err := filepath.Glob(beelogLogs)
	if err != nil {
		return err
	}
	blogs = rmvRepetitiveLogs(blogs)

	// sorts by lenght and lexicographically for equal len
	sort.Sort(byLenAlpha(dlogs))
	sort.Sort(byLenAlpha(blogs))

	//logs := [][]string{dlogs, blogs}
	//names := []string{"disktrad", "beelog"}
	logs := [][]string{blogs}
	names := []string{"beelog"}

	for i, l := range logs {
		state := NewMockState()
		var nCmds uint64
		var totalSize int64

		for _, fn := range l {
			fd, err := os.OpenFile(fn, os.O_RDONLY, 0400)
			if err != nil && err != io.EOF {
				return fmt.Errorf("failed while opening log '%s', err: '%s'", fn, err.Error())
			}

			info, err := fd.Stat()
			if err != nil {
				fd.Close()
				return err
			}
			totalSize += info.Size()

			n, err := state.InstallRecovStateFromReader(fd)
			if err != nil {
				fd.Close()
				return err
			}
			nCmds += n
			fd.Close()
		}

		fmt.Println(
			"=========================",
			"\nFinished installing logs for", names[i],
			"\nNum of diff logs:", len(l),
			"\nNum of commands:", nCmds,
			"\nRepetitive keys: TODO",
			"\nTotal state size (bytes):", totalSize,
		)
	}
	return nil
}

// rmvRepetitiveLogs identifies the first node identifier within log filenames,
// then ignores logs from all different nodes.
func rmvRepetitiveLogs(logs []string) []string {
	if len(logs) < 1 {
		return nil
	}
	uniques := make([]string, 0, len(logs))

	// e.g. beelog-node1.1000.log -> [beelog-node1., 1000.log]
	split := strings.SplitAfterN(logs[0], ".", 2)

	// e.g. beelog-node1. -> node1.
	id := strings.SplitAfter(split[0], "-")[1]

	for _, l := range logs {
		if strings.Contains(l, id) {
			uniques = append(uniques, l)
		}
	}
	return uniques
}

type byLenAlpha []string

func (a byLenAlpha) Len() int      { return len(a) }
func (a byLenAlpha) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byLenAlpha) Less(i, j int) bool {
	// lenght order prio
	if len(a[i]) < len(a[j]) {
		return true
	}
	// alphabetic
	if len(a[i]) == len(a[j]) {
		return strings.Compare(a[i], a[j]) == -1
	}
	return false
}
