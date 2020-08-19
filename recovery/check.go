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
	disktradLogs = "/tmp/log-file*.log"
	beelogLogs   = "/tmp/beelog*.log"
)

func checkLocalLogs() error {
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

	logs := [][]string{dlogs, blogs}
	names := []string{"disktrad", "beelog"}

	for i, l := range logs {
		state := NewMockState()
		var nCmds uint64

		for _, fn := range l {
			fd, err := os.OpenFile(fn, os.O_RDONLY, 0400)
			if err != nil && err != io.EOF {
				return fmt.Errorf("failed while opening log '%s', err: '%s'", fn, err.Error())
			}

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
			"\nNum of commands:", nCmds,
			"\nRepetitive keys: TODO",
			"\nState size (bytes): TODO",
		)
	}
	return nil
}

// TODO: must ignore equal logs from diff nodes...
func rmvRepetitiveLogs(logs []string) []string {
	return nil
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
