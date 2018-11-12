package process

import (
	"bufio"
	"io"
	"sync"

	"github.com/buildkite/agent/logger"
)

// LineScanner is a line-by-line scanner and filter
type LineScanner struct {
	// LineCallback is called with each line provided LineCallbackFilter returns true
	LineCallback func(string)

	// LinePreProcessor filters every line
	LinePreProcessor func(string) string

	// LineCallbackFilter determines whether a line is passed to LineCallback
	LineCallbackFilter func(string) bool

	// LinePostProcessor filters every line before being written to writer
	LinePostProcessor func(string) string
}

// ScanInto reads lines from the reader and writes them out according to filters and processors
func (l *LineScanner) ScanInto(w io.Writer, r io.Reader) error {
	var reader = bufio.NewReader(r)
	var appending []byte
	var lineCallbackWaitGroup sync.WaitGroup

	logger.Debug("[LineScanner] Starting to read lines")

	// Note that we do this manually rather than bufio.Scanner
	// because we need to handle very long lines

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				logger.Debug("[LineScanner] Encountered EOF")
				break
			}
			return err
		}

		// If isPrefix is true, that means we've got a really
		// long line incoming, and we'll keep appending to it
		// until isPrefix is false (which means the long line
		// has ended.
		if isPrefix && appending == nil {
			logger.Debug("[LineScanner] Line is too long to read, going to buffer it until it finishes")

			// bufio.ReadLine returns a slice which is only valid until the next invocation
			// since it points to its own internal buffer array. To accumulate the entire
			// result we make a copy of the first prefix, and ensure there is spare capacity
			// for future appends to minimize the need for resizing on append.
			appending = make([]byte, len(line), (cap(line))*2)
			copy(appending, line)

			continue
		}

		// Should we be appending?
		if appending != nil {
			appending = append(appending, line...)

			// No more isPrefix! Line is finished!
			if !isPrefix {
				logger.Debug("[LineScanner] Finished buffering long line")
				line = appending

				// Reset appending back to nil
				appending = nil
			} else {
				continue
			}
		}

		lineString := string(line)

		// Always apply the line pre-processor if it exists
		if l.LinePreProcessor != nil {
			lineString = l.LinePreProcessor(string(line))
		}

		// Apply the line callback if the filter says so
		if l.LineCallbackFilter != nil && l.LineCallback != nil {
			lineCallbackWaitGroup.Add(1)
			go func(lineString string) {
				defer lineCallbackWaitGroup.Done()
				if l.LineCallbackFilter(lineString) {
					l.LineCallback(lineString)
				}
			}(lineString)
		}

		// Apply post processor if it exists
		if l.LinePostProcessor != nil {
			lineString = l.LinePostProcessor(lineString)
		}

		// Finally write the line to the writer
		w.Write([]byte(lineString + "\n"))
	}

	// We need to make sure all the line callbacks have finish before
	// finish up the process
	logger.Debug("[LineScanner] Waiting for callbacks to finish")
	lineCallbackWaitGroup.Wait()

	logger.Debug("[LineScanner] Finished")
	return nil
}
