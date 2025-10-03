package utils

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/path/to/file", "path/to/file"},
		{"path/to/file", "path/to/file"},
		{"/path/../to/file", "to/file"},
		{"./path/to/file", "path/to/file"},
		{"", "."},
		{"/", "."},
	}

	for _, test := range tests {
		result := NormalizePath(test.input)
		if result != test.expected {
			t.Errorf("NormalizePath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"path/to/file", true},
		{"file.txt", true},
		{"../../../etc/passwd", false},
		{"path/../to/file", false},
		{"path/to/../file", false},
		{"..", false},
		{".", true},
	}

	for _, test := range tests {
		result := IsValidPath(test.input)
		if result != test.valid {
			t.Errorf("IsValidPath(%q) = %v, expected %v", test.input, result, test.valid)
		}
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path/to/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/path/to/file", "file"},
		{"", "."},
	}

	for _, test := range tests {
		result := GetFileName(test.input)
		if result != test.expected {
			t.Errorf("GetFileName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestGetDir(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path/to/file.txt", "path/to"},
		{"file.txt", "."},
		{"/path/to/file", "/path/to"},
	}

	for _, test := range tests {
		result := GetDir(test.input)
		if result != test.expected {
			t.Errorf("GetDir(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"path", "to", "file"}, "path/to/file"},
		{[]string{"/path", "to", "file"}, "path/to/file"},
		{[]string{"path", "..", "file"}, "file"},
	}

	for _, test := range tests {
		result := JoinPath(test.parts...)
		if result != test.expected {
			t.Errorf("JoinPath(%v) = %q, expected %q", test.parts, result, test.expected)
		}
	}
}

func TestIsDir(t *testing.T) {
	tests := []struct {
		input  string
		isDir bool
	}{
		{"path/to/dir/", true},
		{"path/to/file", false},
		{"file.txt", false},
		{"", false},
	}

	for _, test := range tests {
		result := IsDir(test.input)
		if result != test.isDir {
			t.Errorf("IsDir(%q) = %v, expected %v", test.input, result, test.isDir)
		}
	}
}

func TestPathMatchesPrefix(t *testing.T) {
	tests := []struct {
		filePath string
		prefix   string
		matches  bool
	}{
		{"path/to/file", "path", true},
		{"path/to/file", "path/to", true},
		{"path/to/file", "", true},
		{"path/to/file", "other", false},
		{"file", "path", false},
	}

	for _, test := range tests {
		result := PathMatchesPrefix(test.filePath, test.prefix)
		if result != test.matches {
			t.Errorf("PathMatchesPrefix(%q, %q) = %v, expected %v", test.filePath, test.prefix, result, test.matches)
		}
	}
}

func TestIsPasswordError(t *testing.T) {
	if !IsPasswordError(ErrWrongPassword) {
		t.Error("ErrWrongPassword should be a password error")
	}

	if !IsPasswordError(ErrPasswordRequired) {
		t.Error("ErrPasswordRequired should be a password error")
	}

	if IsPasswordError(ErrFileNotFound) {
		t.Error("ErrFileNotFound should not be a password error")
	}
}

func TestIsNotFoundError(t *testing.T) {
	if !IsNotFoundError(ErrFileNotFound) {
		t.Error("ErrFileNotFound should be a not found error")
	}

	if IsNotFoundError(ErrWrongPassword) {
		t.Error("ErrWrongPassword should not be a not found error")
	}
}
