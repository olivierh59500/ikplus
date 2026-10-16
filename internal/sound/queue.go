package sound

import (
	"io"
	"sync"
)

// PCMQueue transfers completed PCM16 stereo frames to the audio backend.
// Reads never call gameplay, rendering or the shared RNG.
type PCMQueue struct {
	mu                           sync.Mutex
	chunks                       [][]byte
	offset, buffered, max, phase int
	padding, closed              bool
}

func NewPCMQueue(maxBytes int) *PCMQueue { return &PCMQueue{max: maxBytes} }

// Push copies the renderer's reused buffer. Old complete chunks are discarded
// if a suspended device falls more than the configured capacity behind.
func (q *PCMQueue) Push(pcm []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || len(pcm) == 0 {
		return
	}
	q.chunks = append(q.chunks, append([]byte(nil), pcm...))
	q.buffered += len(pcm)
	for q.buffered > q.max && len(q.chunks) > 1 {
		index := 0
		if q.offset != 0 {
			index = 1
		}
		q.buffered -= len(q.chunks[index])
		copy(q.chunks[index:], q.chunks[index+1:])
		q.chunks[len(q.chunks)-1] = nil
		q.chunks = q.chunks[:len(q.chunks)-1]
	}
}

func (q *PCMQueue) Buffered() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.buffered
}

// Read supplies silence on an underrun without blocking the backend. A gap
// finishes its stereo frame before new PCM starts, even for odd byte requests.
func (q *PCMQueue) Read(p []byte) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for n < len(p) {
		if q.padding && q.phase != 0 {
			count := min(len(p)-n, 4-q.phase)
			clear(p[n : n+count])
			n += count
			q.phase = (q.phase + count) & 3
			continue
		}
		q.padding = false
		if len(q.chunks) == 0 {
			if q.closed {
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}
			clear(p[n:])
			q.phase = (q.phase + len(p) - n) & 3
			q.padding = true
			return len(p), nil
		}
		count := copy(p[n:], q.chunks[0][q.offset:])
		q.offset += count
		q.buffered -= count
		q.phase = (q.phase + count) & 3
		n += count
		if q.offset == len(q.chunks[0]) {
			q.chunks[0] = nil
			q.chunks = q.chunks[1:]
			q.offset = 0
		}
	}
	return n, nil
}

func (q *PCMQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}
