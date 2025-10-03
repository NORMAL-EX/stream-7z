package formats

import (
	"context"
	"io"
	"time"
)

// FileEntry represents a file within an archive
type FileEntry struct {
	Path           string    // Full path within archive
	Size           int64     // Uncompressed size
	CompressedSize int64     // Compressed size
	ModTime        time.Time // Modification time
	IsDir          bool      // Whether this is a directory
}

// ArchiveInfo contains metadata about an archive
type ArchiveInfo struct {
	IsEncrypted      bool        // Whether the archive contains encrypted files
	RequiresPassword bool        // Whether a password is needed
	TotalFiles       int         // Total number of files (excluding directories)
	TotalSize        int64       // Total uncompressed size
	Files            []FileEntry // List of all files
	Comment          string      // Archive comment (if any)
}

// Format defines the interface that all archive format handlers must implement
type Format interface {
	// Name returns the format name (e.g., "zip", "rar", "7z")
	Name() string

	// Extensions returns the file extensions this format handles
	Extensions() []string

	// Detect checks if the reader contains an archive of this format
	Detect(ctx context.Context, reader io.ReaderAt, size int64) (bool, error)

	// GetInfo retrieves metadata about the archive
	GetInfo(ctx context.Context, reader io.ReaderAt, size int64, password string) (*ArchiveInfo, error)

	// ListFiles returns a list of files in the archive
	// If innerPath is provided, only files within that directory are returned
	ListFiles(ctx context.Context, reader io.ReaderAt, size int64, innerPath string, password string) ([]FileEntry, error)

	// ExtractFile extracts a single file from the archive
	// Returns a reader for the file content and the file size
	ExtractFile(ctx context.Context, reader io.ReaderAt, size int64, filePath string, password string) (io.ReadCloser, int64, error)
}

// Registry holds all registered format handlers
type Registry struct {
	formats map[string]Format
}

// NewRegistry creates a new format registry
func NewRegistry() *Registry {
	return &Registry{
		formats: make(map[string]Format),
	}
}

// Register adds a format handler to the registry
func (r *Registry) Register(format Format) {
	r.formats[format.Name()] = format
}

// Get retrieves a format handler by name
func (r *Registry) Get(name string) (Format, bool) {
	f, ok := r.formats[name]
	return f, ok
}

// DetectFormat attempts to detect the archive format
func (r *Registry) DetectFormat(ctx context.Context, reader io.ReaderAt, size int64, extension string) (Format, error) {
	// First try by extension
	for _, format := range r.formats {
		for _, ext := range format.Extensions() {
			if ext == extension {
				detected, err := format.Detect(ctx, reader, size)
				if err == nil && detected {
					return format, nil
				}
			}
		}
	}

	// Try all formats
	for _, format := range r.formats {
		detected, err := format.Detect(ctx, reader, size)
		if err == nil && detected {
			return format, nil
		}
	}

	return nil, ErrFormatNotDetected
}

// GetAllFormats returns all registered formats
func (r *Registry) GetAllFormats() []Format {
	formats := make([]Format, 0, len(r.formats))
	for _, f := range r.formats {
		formats = append(formats, f)
	}
	return formats
}

// Global registry instance
var globalRegistry = NewRegistry()

// RegisterFormat registers a format in the global registry
func RegisterFormat(format Format) {
	globalRegistry.Register(format)
}

// GetFormat retrieves a format from the global registry
func GetFormat(name string) (Format, bool) {
	return globalRegistry.Get(name)
}

// DetectFormat detects format using the global registry
func DetectFormat(ctx context.Context, reader io.ReaderAt, size int64, extension string) (Format, error) {
	return globalRegistry.DetectFormat(ctx, reader, size, extension)
}

// Common errors
var (
	ErrFormatNotDetected = &FormatError{Message: "unable to detect archive format"}
	ErrPasswordIncorrect = &FormatError{Message: "incorrect password"}
	ErrPasswordRequired  = &FormatError{Message: "password required"}
	ErrFileNotFound      = &FormatError{Message: "file not found in archive"}
	ErrNotSupported      = &FormatError{Message: "operation not supported"}
)

// FormatError represents a format-specific error
type FormatError struct {
	Message string
	Cause   error
}

func (e *FormatError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *FormatError) Unwrap() error {
	return e.Cause
}
