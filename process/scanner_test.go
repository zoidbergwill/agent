package process_test

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buildkite/agent/process"
)

const longTestOutput = `+++ My header
llamas
and more llamas
a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line a very long line
and some alpacas
`

func TestScannerCallsLineCallbacksForEachOutputLine(t *testing.T) {
	var lineCounter int32
	var lines []string
	var linesLock sync.Mutex

	s := process.LineScanner{
		LineCallback: func(s string) {
			linesLock.Lock()
			defer linesLock.Unlock()
			lines = append(lines, s)
		},
		LinePreProcessor: func(s string) string {
			lineNumber := atomic.AddInt32(&lineCounter, 1)
			return fmt.Sprintf("#%d: chars %d", lineNumber, len(s))
		},
		LineCallbackFilter: func(s string) bool {
			return true
		},
	}

	output := scanIntoString(t, longTestOutput, s)

	var expected = []string{
		`#1: chars 13`,
		`#2: chars 6`,
		`#3: chars 15`,
		`#4: chars 237`,
		`#5: chars 16`,
	}

	if !reflect.DeepEqual(expected, lines) {
		t.Fatalf("Lines was unexpected:\nWanted: %v\nGot: %v\n", expected, lines)
	}

	var expectedOutput = strings.Join(expected, "\n") + "\n"

	if output != expectedOutput {
		t.Fatalf("Output was unexpected:\nWanted: %q\nGot: %q\n", expectedOutput, output)
	}
}

func TestScannerCallsPostProcessorCallback(t *testing.T) {
	s := process.LineScanner{
		LinePostProcessor: func(s string) string {
			return fmt.Sprintf("prefix: %s", s)
		},
	}

	input := "line 1\nline 2\n"
	output := scanIntoString(t, input, s)

	expectedOutput := "prefix: line 1\nprefix: line 2\n"
	if expectedOutput != output {
		t.Fatalf("Output was unexpected:\nWanted: %q\nGot: %q\n", expectedOutput, output)
	}
}

func scanIntoString(t *testing.T, str string, s process.LineScanner) string {
	b := &bytes.Buffer{}
	pr, pw := io.Pipe()

	go func() {
		for _, line := range strings.Split(strings.TrimSuffix(str, "\n"), "\n") {
			fmt.Fprintf(pw, "%s\n", line)
			time.Sleep(time.Millisecond * 10)
		}
		pw.Close()
	}()

	if err := s.ScanInto(b, pr); err != nil {
		t.Fatal(err)
	}

	return b.String()
}
