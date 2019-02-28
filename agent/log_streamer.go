package agent

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"

	"github.com/buildkite/agent/logger"
)

type LogStreamerConfig struct {
	// How many log streamer workers are running at any one time
	Concurrency int

	// The maximum size of chunks
	MaxChunkSizeBytes int
}

type LogStreamer struct {
	// The configuration
	conf LogStreamerConfig

	// The logger instance to use
	logger *logger.Logger

	// A counter of how many chunks failed to upload
	chunksFailedCount int32

	// The callback called when a chunk is ready for upload
	callback func(chunk *LogStreamerChunk) error

	// The queue of chunks that are needing to be uploaded
	queue chan *LogStreamerChunk

	// Total size in bytes of the log
	bytes int

	// Each chunk is assigned an order
	order int

	// Every time we add a job to the queue, we increase the wait group
	// queue so when the streamer shuts down, we can block until all work
	// has been added.
	chunkWaitGroup sync.WaitGroup

	// Only allow processing one at a time
	mu sync.Mutex
}

type LogStreamerChunk struct {
	// The contents of the chunk
	Data []byte

	// The sequence number of this chunk
	Order int

	// The byte offset of this chunk
	Offset int

	// The byte size of this chunk
	Size int
}

// Creates a new instance of the log streamer
func NewLogStreamer(l *logger.Logger, cb func(chunk *LogStreamerChunk) error, c LogStreamerConfig) *LogStreamer {
	return &LogStreamer{
		logger:   l,
		conf:     c,
		callback: cb,
		queue:    make(chan *LogStreamerChunk, 1024),
	}
}

// Spins up x number of log streamer workers
func (ls *LogStreamer) Start() error {
	if ls.conf.MaxChunkSizeBytes == 0 {
		return errors.New("Maximum chunk size must be more than 0. No logs will be sent.")
	}

	for i := 0; i < ls.conf.Concurrency; i++ {
		go logStreamerWorker(i, ls)
	}

	return nil
}

func (ls *LogStreamer) FailedChunks() int {
	return int(atomic.LoadInt32(&ls.chunksFailedCount))
}

// Write a chunk of data to the log streamer
func (ls *LogStreamer) Write(blob []byte) (n int, err error) {
	// Serialize parallel writes
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// How many chunks do we have that fit within the MaxChunkSizeBytes?
	numberOfChunks := int(math.Ceil(float64(len(blob)) / float64(ls.conf.MaxChunkSizeBytes)))

	// Increase the wait group by the amount of chunks we're going to add
	ls.chunkWaitGroup.Add(numberOfChunks)

	for i := 0; i < numberOfChunks; i++ {
		// Find the upper limit of the blob
		upperLimit := (i + 1) * ls.conf.MaxChunkSizeBytes
		if upperLimit > len(blob) {
			upperLimit = len(blob)
		}

		// Grab the section of the blob
		partialChunk := blob[i*ls.conf.MaxChunkSizeBytes : upperLimit]

		// Increment the order
		ls.order += 1

		// Create the chunk and append it to our list
		chunk := LogStreamerChunk{
			Data:   partialChunk,
			Order:  ls.order,
			Offset: ls.bytes,
			Size:   len(partialChunk),
		}

		ls.queue <- &chunk

		// Save the new amount of bytes
		ls.bytes += len(partialChunk)
		n += len(partialChunk)
	}

	return n, err
}

// Waits for all the chunks to be uploaded, then shuts down all the workers
func (ls *LogStreamer) Stop() error {
	ls.logger.Debug("[LogStreamer] Waiting for all the chunks to be uploaded")

	ls.chunkWaitGroup.Wait()

	ls.logger.Debug("[LogStreamer] Shutting down all workers")

	for n := 0; n < ls.conf.Concurrency; n++ {
		ls.queue <- nil
	}

	return nil
}

// The actual log streamer worker
func logStreamerWorker(id int, ls *LogStreamer) {
	ls.logger.Debug("[LogStreamer/Worker#%d] Worker is starting...", id)

	var chunk *LogStreamerChunk
	for {
		// Get the next chunk (pointer) from the queue. This will block
		// until something is returned.
		chunk = <-ls.queue

		// If the next chunk is nil, then there is no more work to do
		if chunk == nil {
			break
		}

		// Upload the chunk
		err := ls.callback(chunk)
		if err != nil {
			atomic.AddInt32(&ls.chunksFailedCount, 1)

			ls.logger.Error("Giving up on uploading chunk %d, this will result in only a partial build log on Buildkite", chunk.Order)
		}

		// Signal to the chunkWaitGroup that this one is done
		ls.chunkWaitGroup.Done()
	}

	ls.logger.Debug("[LogStreamer/Worker#%d] Worker has shutdown", id)
}
