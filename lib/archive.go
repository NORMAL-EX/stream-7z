package lib

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/NORMAL-EX/stream-7z/lib/formats"
	"github.com/NORMAL-EX/stream-7z/lib/rangehttp"
	"github.com/NORMAL-EX/stream-7z/lib/utils"
)

// Archive represents a remote archive file accessed via HTTP Range requests
type Archive struct {
	config     *Config
	url        string
	size       int64
	reader     *rangehttp.RangeReader
	format     formats.Format
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *rangehttp.Client
}

// NewArchive creates a new Archive instance from a URL
func NewArchive(archiveURL string, config *Config) (*Archive, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate URL
	parsedURL, err := url.Parse(archiveURL)
	if err != nil {
		return nil, utils.WrapError(utils.ErrInvalidURL, "invalid URL: %s", archiveURL)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, utils.WrapError(utils.ErrInvalidURL, "only HTTP/HTTPS URLs are supported")
	}

	// Create HTTP client
	httpClient := rangehttp.NewClient(
		config.HTTPClient,
		config.Headers,
		config.UserAgent,
		config.Timeout,
	)

	// Create context with timeout from config
	// If timeout is negative, no timeout is set (unlimited)
	var ctx context.Context
	var cancel context.CancelFunc
	
	if config.Timeout < 0 {
		// Negative timeout means no timeout limit
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		timeout := config.Timeout
		if timeout == 0 {
			timeout = 120 * time.Second // Default 120 seconds
		}
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	}

	// Get file size and check Range support
	size, supportsRange, err := httpClient.HeadRequest(ctx, archiveURL)
	if err != nil {
		cancel()
		return nil, utils.WrapError(err, "failed to get file information")
	}

	if !supportsRange && config.Debug {
		fmt.Printf("Warning: Server does not support Range requests, performance may be degraded\n")
	}

	// Check max file size
	if config.MaxFileSize > 0 && size > config.MaxFileSize {
		cancel()
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", size, config.MaxFileSize)
	}

	// Create range reader
	rangeReader, err := rangehttp.NewRangeReader(ctx, httpClient, archiveURL, size)
	if err != nil {
		cancel()
		return nil, utils.WrapError(err, "failed to create range reader")
	}

	// Detect format
	ext := strings.ToLower(path.Ext(parsedURL.Path))
	format, err := formats.DetectFormat(ctx, rangeReader, size, ext)
	if err != nil {
		rangeReader.Close()
		cancel()
		return nil, utils.WrapError(utils.ErrUnsupportedFormat, "unable to detect archive format")
	}

	return &Archive{
		config:     config,
		url:        archiveURL,
		size:       size,
		reader:     rangeReader,
		format:     format,
		ctx:        ctx,
		cancel:     cancel,
		httpClient: httpClient,
	}, nil
}

// GetInfo returns metadata about the archive
func (a *Archive) GetInfo(password string) (*formats.ArchiveInfo, error) {
	return a.format.GetInfo(a.ctx, a.reader, a.size, password)
}

// ListFiles returns a list of files in the archive
// If innerPath is empty, returns root level files
// If innerPath is specified, returns files within that directory
func (a *Archive) ListFiles(innerPath string, password string) ([]formats.FileEntry, error) {
	return a.format.ListFiles(a.ctx, a.reader, a.size, innerPath, password)
}

// ExtractFile extracts a single file from the archive
// Returns a reader for the file content
func (a *Archive) ExtractFile(filePath string, password string) (io.ReadCloser, int64, error) {
	// Validate path
	if !utils.IsValidPath(filePath) {
		return nil, 0, utils.ErrPathTraversal
	}

	return a.format.ExtractFile(a.ctx, a.reader, a.size, filePath, password)
}

// Close closes the archive and releases resources
func (a *Archive) Close() error {
	if a.reader != nil {
		a.reader.Close()
	}
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// URL returns the archive URL
func (a *Archive) URL() string {
	return a.url
}

// Size returns the archive size in bytes
func (a *Archive) Size() int64 {
	return a.size
}

// Format returns the detected archive format name
func (a *Archive) Format() string {
	if a.format != nil {
		return a.format.Name()
	}
	return "unknown"
}

// QuickInfo is a convenience function that creates an Archive, gets info, and closes it
func QuickInfo(archiveURL string, password string, config *Config) (*formats.ArchiveInfo, error) {
	archive, err := NewArchive(archiveURL, config)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	return archive.GetInfo(password)
}

// QuickList is a convenience function that creates an Archive, lists files, and closes it
func QuickList(archiveURL string, innerPath string, password string, config *Config) ([]formats.FileEntry, error) {
	archive, err := NewArchive(archiveURL, config)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	return archive.ListFiles(innerPath, password)
}

// QuickExtract is a convenience function that creates an Archive, extracts a file, and closes the archive
// Note: The returned ReadCloser must still be closed by the caller
func QuickExtract(archiveURL string, filePath string, password string, config *Config) (io.ReadCloser, int64, error) {
	archive, err := NewArchive(archiveURL, config)
	if err != nil {
		return nil, 0, err
	}

	reader, size, err := archive.ExtractFile(filePath, password)
	if err != nil {
		archive.Close()
		return nil, 0, err
	}

	// Return a wrapped reader that closes the archive when the file reader is closed
	return &archiveReader{
		ReadCloser: reader,
		archive:    archive,
	}, size, nil
}

// archiveReader wraps a file reader and ensures the archive is closed
type archiveReader struct {
	io.ReadCloser
	archive *Archive
	closed  bool
}

func (ar *archiveReader) Close() error {
	if ar.closed {
		return nil
	}
	ar.closed = true

	// Close the file reader first
	err1 := ar.ReadCloser.Close()

	// Then close the archive
	err2 := ar.archive.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

// WithContext creates a new Archive with a custom context
func NewArchiveWithContext(ctx context.Context, archiveURL string, config *Config) (*Archive, error) {
	archive, err := NewArchive(archiveURL, config)
	if err != nil {
		return nil, err
	}

	// Don't cancel the old context - RangeReader is still using it!
	// Instead, create a child context from the provided context
	oldCancel := archive.cancel
	archive.ctx, archive.cancel = context.WithCancel(ctx)
	
	// Keep the old context alive by not calling oldCancel immediately
	// It will be cleaned up when the archive is closed
	_ = oldCancel

	return archive, nil
}

// WithTimeout creates a new Archive with a timeout
func NewArchiveWithTimeout(archiveURL string, timeout time.Duration, config *Config) (*Archive, error) {
	archive, err := NewArchive(archiveURL, config)
	if err != nil {
		return nil, err
	}

	// Replace context with timeout
	archive.cancel()
	archive.ctx, archive.cancel = context.WithTimeout(context.Background(), timeout)

	return archive, nil
}
