package agent

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/buildkite/agent/logger"
)

func TestLogStreamer(t *testing.T) {
	buf := &bytes.Buffer{}
	var bufMu sync.Mutex

	uploader := func(chunk *LogStreamerChunk) error {
		bufMu.Lock()
		defer bufMu.Unlock()
		if _, err := buf.Write(chunk.Data); err != nil {
			return err
		}
		return nil
	}

	streamer := NewLogStreamer(logger.Discard, uploader, LogStreamerConfig{
		Concurrency:       1,
		MaxChunkSizeBytes: 2,
	})

	if err := streamer.Start(); err != nil {
		t.Fatal(err)
	}

	var expected string
	for i := 0; i < 1000; i++ {
		s := strings.Repeat("llamas", 10)
		expected += s
		if _, err := streamer.Write([]byte(s)); err != nil {
			t.Fatal(err)
		}
	}

	if err := streamer.Stop(); err != nil {
		t.Fatal(err)
	}

	bufMu.Lock()
	defer bufMu.Unlock()

	if expected != buf.String() {
		t.Fatalf("Bad output: %q expected %q", buf.String(), expected)
	}
}

func TestLogStreamerWithConcurrency(t *testing.T) {
	var chunks []LogStreamerChunk
	var chunksMu sync.Mutex

	uploader := func(chunk *LogStreamerChunk) error {
		chunksMu.Lock()
		defer chunksMu.Unlock()
		chunks = append(chunks, *chunk)
		return nil
	}

	streamer := NewLogStreamer(logger.Discard, uploader, LogStreamerConfig{
		Concurrency:       3,
		MaxChunkSizeBytes: 2,
	})

	if err := streamer.Start(); err != nil {
		t.Fatal(err)
	}

	if _, err := streamer.Write([]byte("llamas\n")); err != nil {
		t.Fatal(err)
	}

	if _, err := streamer.Write([]byte("alpaca\n")); err != nil {
		t.Fatal(err)
	}

	if err := streamer.Stop(); err != nil {
		t.Fatal(err)
	}

	chunksMu.Lock()
	defer chunksMu.Unlock()

	if l := len(chunks); l != 8 {
		t.Fatalf("Bad number of chunks, got %d", l)
	}

	// Sort by order
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Order < chunks[j].Order
	})

	for idx, expected := range []LogStreamerChunk{
		{Data: []byte("ll"), Order: 1, Offset: 0, Size: 2},
		{Data: []byte("am"), Order: 2, Offset: 2, Size: 2},
		{Data: []byte("as"), Order: 3, Offset: 4, Size: 2},
		{Data: []byte("\n"), Order: 4, Offset: 6, Size: 1},
		{Data: []byte("al"), Order: 5, Offset: 7, Size: 2},
		{Data: []byte("pa"), Order: 6, Offset: 9, Size: 2},
		{Data: []byte("ca"), Order: 7, Offset: 11, Size: 2},
		{Data: []byte("\n"), Order: 8, Offset: 13, Size: 1},
	} {
		if !reflect.DeepEqual(expected, chunks[idx]) {
			t.Fatalf("Bad chunk at idx %d: %#v, expected %#v",
				idx, chunks[idx], expected)
		}
	}
}

func BenchmarkLogStreamer(b *testing.B) {
	uploader := func(chunk *LogStreamerChunk) error {
		return nil
	}

	streamer := NewLogStreamer(logger.Discard, uploader, LogStreamerConfig{
		Concurrency:       1,
		MaxChunkSizeBytes: 2,
	})

	if err := streamer.Start(); err != nil {
		b.Fatal(err)
	}

	payload := []byte(strings.Repeat("llamas", 1000))

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		streamer.Write(payload)
	}
}
