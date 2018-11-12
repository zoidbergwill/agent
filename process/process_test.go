package process_test

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/buildkite/agent/logger"
	"github.com/buildkite/agent/process"
)

func TestProcessRunsAndCallsStartCallback(t *testing.T) {
	var started int32

	p := process.Process{
		Script: []string{os.Args[0]},
		Env:    []string{"TEST_MAIN=tester"},
		StartCallback: func() {
			atomic.AddInt32(&started, 1)
		},
	}

	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	if startedVal := atomic.LoadInt32(&started); startedVal != 1 {
		t.Fatalf("Expected started to be 1, got %d", startedVal)
	}

	if exitStatus := p.ExitStatus; exitStatus != "0" {
		t.Fatalf("Expected ExitStatus of 0, got %v", exitStatus)
	}

	output := p.Output()
	if output != string(longTestOutput) {
		t.Fatalf("Output was unexpected:\nWanted: %q\nGot:    %q\n", longTestOutput, output)
	}
}

func TestProcessOutputIsSafeFromRaces(t *testing.T) {
	var counter int32

	p := process.Process{
		Script: []string{os.Args[0]},
		Env:    []string{"TEST_MAIN=tester"},
	}

	// the job_runner has a for loop that calls IsRunning and Output, so this checks those are safe from races
	p.StartCallback = func() {
		for p.IsRunning() {
			_ = p.Output()
			atomic.AddInt32(&counter, 1)
			time.Sleep(time.Millisecond * 10)
		}
	}

	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	output := p.Output()
	if output != string(longTestOutput) {
		t.Fatalf("Output was unexpected:\nWanted: %q\nGot:    %q\n", longTestOutput, output)
	}

	if counterVal := atomic.LoadInt32(&counter); counterVal < 10 {
		t.Fatalf("Expected counter to be at least 10, got %d", counterVal)
	}
}

func TestKillingProcess(t *testing.T) {
	logger.SetLevel(logger.DEBUG)

	p := process.Process{
		Script: []string{os.Args[0]},
		Env:    []string{"TEST_MAIN=tester-signal"},
	}

	var wg sync.WaitGroup
	wg.Add(1)

	p.StartCallback = func() {
		go func() {
			<-time.After(time.Millisecond * 10)
			if err := p.Kill(); err != nil {
				t.Error(err)
			}
		}()
	}

	go func() {
		defer wg.Done()
		if err := p.Start(); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()

	output := p.Output()
	if output != `SIG terminated` {
		t.Fatalf("Bad output: %q", output)
	}
}

// Invoked by `go test`, switch between helper and running tests based on env
func TestMain(m *testing.M) {
	switch os.Getenv("TEST_MAIN") {
	case "tester":
		for _, line := range strings.Split(strings.TrimSuffix(longTestOutput, "\n"), "\n") {
			fmt.Printf("%s\n", line)
			time.Sleep(time.Millisecond * 20)
		}
		os.Exit(0)

	case "tester-signal":
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGINT,
		)

		sig := <-signals
		fmt.Printf("SIG %v", sig)
		os.Exit(0)

	default:
		os.Exit(m.Run())
	}
}
