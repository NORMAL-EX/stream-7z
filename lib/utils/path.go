package utils

import (
	"path"
	"strings"
)

// NormalizePath normalizes archive file paths
// Removes leading slashes and ensures consistent separators
func NormalizePath(p string) string {
	// Remove leading slash
	p = strings.TrimPrefix(p, "/")
	// Clean path
	p = path.Clean(p)
	// Ensure no leading slash after clean
	p = strings.TrimPrefix(p, "/")
	return p
}

// IsValidPath checks if a path is valid and not attempting path traversal
func IsValidPath(p string) bool {
	// Check for path traversal attempts
	if strings.Contains(p, "..") {
		return false
	}
	// Normalize and check if it changed (indicates suspicious path)
	normalized := NormalizePath(p)
	if strings.Contains(normalized, "..") {
		return false
	}
	return true
}

// GetFileName extracts the file name from a path
func GetFileName(p string) string {
	return path.Base(p)
}

// GetDir extracts the directory from a path
func GetDir(p string) string {
	return path.Dir(p)
}

// JoinPath joins path segments safely
func JoinPath(elem ...string) string {
	joined := path.Join(elem...)
	return NormalizePath(joined)
}

// IsDir checks if a path represents a directory
func IsDir(p string) bool {
	return strings.HasSuffix(p, "/")
}

// PathMatchesPrefix checks if path starts with prefix
func PathMatchesPrefix(filePath, prefix string) bool {
	filePath = NormalizePath(filePath)
	prefix = NormalizePath(prefix)
	
	if prefix == "" {
		return true
	}
	
	return strings.HasPrefix(filePath, prefix+"/") || filePath == prefix
}
