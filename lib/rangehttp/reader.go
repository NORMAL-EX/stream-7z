package rangehttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// RangeReader provides io.ReaderAt interface using HTTP Range requests
type RangeReader struct {
	ctx        context.Context
	cancel     context.CancelFunc
	client     *Client
	url        string
	size       int64
	mu         sync.Mutex
	activeReqs map[int64]io.ReadCloser // Track active readers by offset
	closed     bool
}

// NewRangeReader creates a new RangeReader for the given URL
func NewRangeReader(ctx context.Context, client *Client, url string, size int64) (*RangeReader, error) {
	if size <= 0 {
		// Try to get size via HEAD request
		headSize, _, err := client.HeadRequest(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed to determine file size: %w", err)
		}
		size = headSize
	}

	childCtx, cancel := context.WithCancel(ctx)

	return &RangeReader{
		ctx:        childCtx,
		cancel:     cancel,
		client:     client,
		url:        url,
		size:       size,
		activeReqs: make(map[int64]io.ReadCloser),
	}, nil
}

// ReadAt reads len(p) bytes starting at offset off
func (r *RangeReader) ReadAt(p []byte, off int64) (n int, err error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, errors.New("reader is closed")
	}
	r.mu.Unlock()

	if off < 0 {
		return 0, errors.New("negative offset")
	}

	if off >= r.size {
		return 0, io.EOF
	}

	// Calculate read length
	length := int64(len(p))
	if off+length > r.size {
		length = r.size - off
	}

	// Perform range request
	reader, err := r.client.RangeRequest(r.ctx, r.url, off, length)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	// Read data
	total := 0
	for total < int(length) {
		nn, err := reader.Read(p[total:])
		total += nn
		if err != nil {
			if err == io.EOF && total == int(length) {
				return total, nil
			}
			return total, err
		}
	}

	return total, nil
}

// Size returns the total size of the remote file
func (r *RangeReader) Size() int64 {
	return r.size
}

// Seek implements io.Seeker (for compatibility)
// Note: This doesn't maintain any internal offset state
func (r *RangeReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 || offset > r.size {
			return 0, errors.New("invalid offset")
		}
		return offset, nil
	case io.SeekEnd:
		if offset > 0 || offset < -r.size {
			return 0, errors.New("invalid offset")
		}
		return r.size + offset, nil
	default:
		return 0, errors.New("invalid whence")
	}
}

// Close closes the RangeReader and cancels any ongoing requests
func (r *RangeReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	r.cancel()

	// Close all active readers
	for _, reader := range r.activeReqs {
		reader.Close()
	}
	r.activeReqs = nil

	return nil
}

// SectionReader creates an io.SectionReader for a portion of the file
type SectionReader struct {
	r       *RangeReader
	off     int64
	limit   int64
	current int64
	mu      sync.Mutex
}

// NewSectionReader creates a new SectionReader
func NewSectionReader(r *RangeReader, off, n int64) *SectionReader {
	return &SectionReader{
		r:       r,
		off:     off,
		limit:   off + n,
		current: off,
	}
}

// Read implements io.Reader
func (s *SectionReader) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current >= s.limit {
		return 0, io.EOF
	}

	if max := s.limit - s.current; int64(len(p)) > max {
		p = p[0:max]
	}

	n, err = s.r.ReadAt(p, s.current)
	s.current += int64(n)
	return
}

// ReadAt implements io.ReaderAt
func (s *SectionReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.limit-s.off {
		return 0, io.EOF
	}

	off += s.off
	if max := s.limit - off; int64(len(p)) > max {
		p = p[0:max]
		n, err = s.r.ReadAt(p, off)
		if err == nil {
			err = io.EOF
		}
		return n, err
	}

	return s.r.ReadAt(p, off)
}

// Seek implements io.Seeker
func (s *SectionReader) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch whence {
	case io.SeekStart:
		offset += s.off
	case io.SeekCurrent:
		offset += s.current
	case io.SeekEnd:
		offset += s.limit
	default:
		return 0, errors.New("invalid whence")
	}

	if offset < s.off {
		return 0, errors.New("seek before start")
	}

	if offset > s.limit {
		return 0, errors.New("seek beyond end")
	}

	s.current = offset
	return offset - s.off, nil
}

// Size returns the size of the section
func (s *SectionReader) Size() int64 {
	return s.limit - s.off
}
