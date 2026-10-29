package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	_ = w.Close()
	os.Stdout = old
	return <-outC
}

func TestPrintUsageHelp_AuthorName(t *testing.T) {
	out := captureStdout(func() {
		printUsageHelp()
	})

	if !strings.Contains(out, "Stephen Brodigan") {
		t.Errorf("expected printUsageHelp to output author Stephen Brodigan, got:\n%s", out)
	}

	if !strings.Contains(out, "Usage:") {
		t.Errorf("expected printUsageHelp to output usage syntax")
	}

	if !strings.Contains(out, "watch") || !strings.Contains(out, "live") {
		t.Errorf("expected printUsageHelp to list commands options")
	}
}
