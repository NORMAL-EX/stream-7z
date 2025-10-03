package utils

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidURL indicates the provided URL is not valid or not supported
	ErrInvalidURL = errors.New("invalid or unsupported URL")

	// ErrUnsupportedFormat indicates the archive format is not supported
	ErrUnsupportedFormat = errors.New("unsupported archive format")

	// ErrWrongPassword indicates the password for encrypted archive is incorrect
	ErrWrongPassword = errors.New("incorrect password for encrypted archive")

	// ErrFileNotFound indicates the requested file was not found in the archive
	ErrFileNotFound = errors.New("file not found in archive")

	// ErrRangeNotSupported indicates the server does not support HTTP Range requests
	ErrRangeNotSupported = errors.New("server does not support range requests")

	// ErrArchiveCorrupted indicates the archive file appears to be corrupted
	ErrArchiveCorrupted = errors.New("archive file is corrupted or invalid")

	// ErrPasswordRequired indicates a password is required but not provided
	ErrPasswordRequired = errors.New("password required for encrypted archive")

	// ErrContextCanceled indicates the operation was canceled via context
	ErrContextCanceled = errors.New("operation canceled")

	// ErrTimeout indicates the operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrPathTraversal indicates an attempt to access files outside archive
	ErrPathTraversal = errors.New("path traversal detected")
)

// WrapError wraps an error with additional context
func WrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// IsPasswordError checks if the error is related to password issues
func IsPasswordError(err error) bool {
	return errors.Is(err, ErrWrongPassword) || errors.Is(err, ErrPasswordRequired)
}

// IsNotFoundError checks if the error is related to file not found
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrFileNotFound)
}
