package rangehttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NORMAL-EX/stream-7z/lib/utils"
)

// Client provides HTTP Range request capabilities
type Client struct {
	httpClient *http.Client
	headers    map[string]string
	userAgent  string
	timeout    time.Duration
	mu         sync.RWMutex
}

// NewClient creates a new Range HTTP client
func NewClient(httpClient *http.Client, headers map[string]string, userAgent string, timeout time.Duration) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	headersCopy := make(map[string]string)
	for k, v := range headers {
		headersCopy[k] = v
	}

	return &Client{
		httpClient: httpClient,
		headers:    headersCopy,
		userAgent:  userAgent,
		timeout:    timeout,
	}
}

// RangeRequest performs a Range HTTP request
func (c *Client) RangeRequest(ctx context.Context, url string, start, length int64) (io.ReadCloser, error) {
	if length == 0 {
		return io.NopCloser(strings.NewReader("")), nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create HTTP request")
	}

	// Set headers
	c.mu.RLock()
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	c.mu.RUnlock()

	// Set Range header
	if length > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, start+length-1))
	} else {
		// length == -1 means read to end
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, utils.WrapError(err, "HTTP request failed")
	}

	// Check status code
	if resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK {
		// Some servers return 200 OK instead of 206 Partial Content
		// We need to verify the Content-Range header
		if resp.StatusCode == http.StatusOK && start > 0 {
			// Server doesn't support range requests
			// We need to discard the bytes before start
			if length == -1 {
				length = resp.ContentLength - start
			}
			return &skipReader{
				reader: resp.Body,
				skip:   start,
				length: length,
			}, nil
		}
		return resp.Body, nil
	}

	resp.Body.Close()
	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// HeadRequest performs a HEAD request to get file size and check Range support
func (c *Client) HeadRequest(ctx context.Context, url string) (size int64, supportsRange bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, false, utils.WrapError(err, "failed to create HEAD request")
	}

	// Set headers
	c.mu.RLock()
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	c.mu.RUnlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, utils.WrapError(err, "HEAD request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get content length
	size = resp.ContentLength
	if size < 0 {
		// Try to parse from Content-Length header
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			size, _ = strconv.ParseInt(cl, 10, 64)
		}
	}

	// Check if server supports range requests
	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange = acceptRanges == "bytes"

	return size, supportsRange, nil
}

// SetHeader sets a custom header
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// SetHeaders sets multiple headers
func (c *Client) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range headers {
		c.headers[k] = v
	}
}

// skipReader wraps a reader to skip initial bytes
// Used when server doesn't support range requests
type skipReader struct {
	reader   io.ReadCloser
	skip     int64
	length   int64
	skipped  int64
	read     int64
	skipOnce sync.Once
}

func (s *skipReader) Read(p []byte) (n int, err error) {
	// Skip initial bytes on first read
	s.skipOnce.Do(func() {
		_, err = io.CopyN(io.Discard, s.reader, s.skip)
		if err != nil {
			return
		}
		s.skipped = s.skip
	})

	if err != nil {
		return 0, err
	}

	// Check if we've read enough
	if s.length > 0 && s.read >= s.length {
		return 0, io.EOF
	}

	// Limit read size if needed
	if s.length > 0 {
		remaining := s.length - s.read
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}

	n, err = s.reader.Read(p)
	s.read += int64(n)
	return n, err
}

func (s *skipReader) Close() error {
	return s.reader.Close()
}
