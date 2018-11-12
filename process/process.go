package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/buildkite/agent/logger"
)

type Process struct {
	Pid        int
	PTY        bool
	Timestamp  bool
	Script     []string
	Env        []string
	ExitStatus string

	// LineScanner is an optional output line processor and filter
	LineScanner *LineScanner

	buffer  outputBuffer
	command *exec.Cmd

	// This callback is called when the process offically starts
	StartCallback func()

	// Running is stored as an int32 so we can use atomic operations to
	// set/get it (it's accessed by multiple goroutines)
	running int32

	mu   sync.Mutex
	done chan struct{}
}

// Start executes the command and blocks until it finishes
func (p *Process) Start() error {
	if p.IsRunning() {
		return fmt.Errorf("Process is already running")
	}

	p.command = exec.Command(p.Script[0], p.Script[1:]...)

	// Create a channel that we use for signaling when the process is
	// done for Done()
	p.mu.Lock()
	if p.done == nil {
		p.done = make(chan struct{})
	}
	p.mu.Unlock()

	// Copy the current processes ENV and merge in the new ones. We do this
	// so the sub process gets PATH and stuff. We merge our path in over
	// the top of the current one so the ENV from Buildkite and the agent
	// take precedence over the agent
	currentEnv := os.Environ()
	p.command.Env = append(currentEnv, p.Env...)

	var waitGroup sync.WaitGroup

	lineReaderPipe, lineWriterPipe := io.Pipe()

	// Toggle between running in a pty
	if p.PTY {
		pty, err := StartPTY(p.command)
		if err != nil {
			p.ExitStatus = "1"
			return err
		}

		p.Pid = p.command.Process.Pid
		p.setRunning(true)

		waitGroup.Add(1)

		go func() {
			logger.Debug("[Process] Starting to copy PTY to the buffer")

			// Copy the pty to our buffer. This will block until it
			// EOF's or something breaks.
			_, err = io.Copy(lineWriterPipe, pty)
			if e, ok := err.(*os.PathError); ok && e.Err == syscall.EIO {
				// We can safely ignore this error, because
				// it's just the PTY telling us that it closed
				// successfully.  See:
				// https://github.com/buildkite/agent/pull/34#issuecomment-46080419
				err = nil
			}

			if err != nil {
				logger.Error("[Process] PTY output copy failed with error: %T: %v", err, err)
			} else {
				logger.Debug("[Process] PTY has finished being copied to the buffer")
			}

			waitGroup.Done()
		}()
	} else {
		p.command.Stdout = lineWriterPipe
		p.command.Stderr = lineWriterPipe
		p.command.Stdin = nil

		err := p.command.Start()
		if err != nil {
			p.ExitStatus = "1"
			return err
		}

		p.Pid = p.command.Process.Pid
		p.setRunning(true)
	}

	logger.Info("[Process] Process is running with PID: %d", p.Pid)

	if p.LineScanner != nil {
		// Add the scanner the waitGroup
		waitGroup.Add(1)

		// Start the Scanner
		go func() {
			defer waitGroup.Done()
			if err := p.LineScanner.ScanInto(&p.buffer, lineReaderPipe); err != nil {
				logger.Error("[Process] Scanner failed with %v", err)
			}
		}()
	} else {
		go io.Copy(&p.buffer, lineReaderPipe)
	}

	// Call the StartCallback
	go p.StartCallback()

	// Wait until the process has finished. The returned error is nil if the command runs,
	// has no problems copying stdin, stdout, and stderr, and exits with a zero exit status.
	waitResult := p.command.Wait()

	// Close the line writer pipe
	lineWriterPipe.Close()

	// The process is no longer running at this point
	p.setRunning(false)

	// Signal waiting consumers in Done() by closing the done channel
	close(p.done)

	// Find the exit status of the script
	p.ExitStatus = getExitStatus(waitResult)

	logger.Info("Process with PID: %d finished with Exit Status: %s", p.Pid, p.ExitStatus)

	// Sometimes (in docker containers) io.Copy never seems to finish. This is a mega
	// hack around it. If it doesn't finish after 1 second, just continue.
	logger.Debug("[Process] Waiting for routines to finish")
	err := timeoutWait(&waitGroup)
	if err != nil {
		logger.Debug("[Process] Timed out waiting for wait group: (%T: %v)", err, err)
	}

	// No error occurred so we can return nil
	return nil
}

// Output returns the current state of the output buffer and can be called incrementally
func (p *Process) Output() string {
	return p.buffer.String()
}

// Done returns a channel that is closed when the process finishes
func (p *Process) Done() <-chan struct{} {
	p.mu.Lock()
	// We create this here in case this is called before Start()
	if p.done == nil {
		p.done = make(chan struct{})
	}
	d := p.done
	p.mu.Unlock()
	return d
}

// Kill terminates the process gracefully. Initially a SIGTERM is sent, and
// then 10 seconds later a SIGTERM is sent.
func (p *Process) Kill() error {
	var err error
	if runtime.GOOS == "windows" {
		// Sending Interrupt on Windows is not implemented.
		// https://golang.org/src/os/exec.go?s=3842:3884#L110
		err = exec.Command("CMD", "/C", "TASKKILL", "/F", "/T", "/PID", strconv.Itoa(p.Pid)).Run()
	} else {
		// Send a sigterm
		err = p.signal(syscall.SIGTERM)
	}
	if err != nil {
		return err
	}

	select {
	// Was successfully terminated
	case <-p.Done():
		logger.Debug("[Process] Process with PID: %d has exited.", p.Pid)

	// Forcefully kill the process after 10 seconds
	case <-time.After(10 * time.Second):
		if err = p.signal(syscall.SIGKILL); err != nil {
			return err
		}
	}

	return nil
}

func (p *Process) signal(sig os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.command != nil && p.command.Process != nil {
		logger.Debug("[Process] Sending signal: %s to PID: %d", sig.String(), p.Pid)

		err := p.command.Process.Signal(sig)
		if err != nil {
			logger.Error("[Process] Failed to send signal: %s to PID: %d (%T: %v)", sig.String(), p.Pid, err, err)
			return err
		}
	} else {
		logger.Debug("[Process] No process to signal yet")
	}

	return nil
}

// Returns whether or not the process is running
// Deprecated: use Done() instead
func (p *Process) IsRunning() bool {
	return atomic.LoadInt32(&p.running) != 0
}

// Sets the running flag of the process
func (p *Process) setRunning(r bool) {
	// Use the atomic package to avoid race conditions when setting the
	// `running` value from multiple routines
	if r {
		atomic.StoreInt32(&p.running, 1)
	} else {
		atomic.StoreInt32(&p.running, 0)
	}
}

// https://github.com/hnakamur/commango/blob/fe42b1cf82bf536ce7e24dceaef6656002e03743/os/executil/executil.go#L29
// TODO: Can this be better?
func getExitStatus(waitResult error) string {
	exitStatus := -1

	if waitResult != nil {
		if err, ok := waitResult.(*exec.ExitError); ok {
			if s, ok := err.Sys().(syscall.WaitStatus); ok {
				exitStatus = s.ExitStatus()
			} else {
				logger.Error("[Process] Unimplemented for system where exec.ExitError.Sys() is not syscall.WaitStatus.")
			}
		} else {
			logger.Error("[Process] Unexpected error type in getExitStatus: %#v", waitResult)
		}
	} else {
		exitStatus = 0
	}

	return fmt.Sprintf("%d", exitStatus)
}

func timeoutWait(waitGroup *sync.WaitGroup) error {
	// Make a chanel that we'll use as a timeout
	c := make(chan int, 1)

	// Start waiting for the routines to finish
	go func() {
		waitGroup.Wait()
		c <- 1
	}()

	select {
	case _ = <-c:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("Timeout")
	}
}

// outputBuffer is a goroutine safe bytes.Buffer
type outputBuffer struct {
	sync.RWMutex
	buf bytes.Buffer
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (ob *outputBuffer) Write(p []byte) (n int, err error) {
	ob.Lock()
	defer ob.Unlock()
	return ob.buf.Write(p)
}

// WriteString appends the contents of s to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (ob *outputBuffer) WriteString(s string) (n int, err error) {
	return ob.Write([]byte(s))
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (ob *outputBuffer) String() string {
	ob.RLock()
	defer ob.RUnlock()
	return ob.buf.String()
}
