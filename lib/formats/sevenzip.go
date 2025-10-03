package formats

import (
	"context"
	"io"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib/utils"
	"github.com/bodgit/sevenzip"
)

// SevenZipFormat handles 7z archives
type SevenZipFormat struct{}

// NewSevenZipFormat creates a new 7z format handler
func NewSevenZipFormat() *SevenZipFormat {
	return &SevenZipFormat{}
}

// Name returns the format name
func (s *SevenZipFormat) Name() string {
	return "7z"
}

// Extensions returns supported file extensions
func (s *SevenZipFormat) Extensions() []string {
	return []string{".7z"}
}

// Detect checks if the reader contains a 7z archive
func (s *SevenZipFormat) Detect(ctx context.Context, reader io.ReaderAt, size int64) (bool, error) {
	// Check 7z magic number
	magic := make([]byte, 6)
	if _, err := reader.ReadAt(magic, 0); err != nil {
		return false, err
	}

	// 7z files start with: 0x37 0x7A 0xBC 0xAF 0x27 0x1C
	if magic[0] == 0x37 && magic[1] == 0x7A && magic[2] == 0xBC &&
		magic[3] == 0xAF && magic[4] == 0x27 && magic[5] == 0x1C {
		return true, nil
	}

	return false, nil
}

// GetInfo retrieves metadata about the 7z archive
func (s *SevenZipFormat) GetInfo(ctx context.Context, reader io.ReaderAt, size int64, password string) (*ArchiveInfo, error) {
	var szReader *sevenzip.Reader
	var err error

	if password != "" {
		szReader, err = sevenzip.NewReaderWithPassword(reader, size, password)
	} else {
		szReader, err = sevenzip.NewReader(reader, size)
	}

	if err != nil {
		// Check if error is due to password
		if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
			info := &ArchiveInfo{
				IsEncrypted:      true,
				RequiresPassword: true,
				Files:            make([]FileEntry, 0),
			}
			if password != "" {
				return info, ErrPasswordIncorrect
			}
			return info, ErrPasswordRequired
		}
		return nil, utils.WrapError(err, "failed to open 7z archive")
	}

	info := &ArchiveInfo{
		IsEncrypted:      false,
		RequiresPassword: false,
		TotalFiles:       0,
		TotalSize:        0,
		Files:            make([]FileEntry, 0),
	}

	for _, file := range szReader.File {
		// Check if file is encrypted
		// Note: 7z library doesn't provide direct encrypted flag
		// We detect it by attempting to open
		isEncrypted := false
		if password == "" {
			// Try to open without password
			rc, err := file.Open()
			if err != nil && (strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted")) {
				isEncrypted = true
				info.IsEncrypted = true
				info.RequiresPassword = true
			} else if rc != nil {
				rc.Close()
			}
		}

		entry := FileEntry{
			Path:           file.Name,
			Size:           int64(file.UncompressedSize),
			CompressedSize: 0, // 7z doesn't provide individual compressed size
			ModTime:        file.Modified,
			IsDir:          file.FileInfo().IsDir(),
		}

		info.Files = append(info.Files, entry)

		if !entry.IsDir {
			info.TotalFiles++
			info.TotalSize += entry.Size
		}

		if isEncrypted {
			break // Stop if we find encrypted content and no password
		}
	}

	if info.IsEncrypted && password == "" {
		return info, ErrPasswordRequired
	}

	return info, nil
}

// ListFiles returns a list of files in the 7z archive
func (s *SevenZipFormat) ListFiles(ctx context.Context, reader io.ReaderAt, size int64, innerPath string, password string) ([]FileEntry, error) {
	var szReader *sevenzip.Reader
	var err error

	if password != "" {
		szReader, err = sevenzip.NewReaderWithPassword(reader, size, password)
	} else {
		szReader, err = sevenzip.NewReader(reader, size)
	}

	if err != nil {
		if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
			if password != "" {
				return nil, ErrPasswordIncorrect
			}
			return nil, ErrPasswordRequired
		}
		return nil, utils.WrapError(err, "failed to open 7z archive")
	}

	innerPath = utils.NormalizePath(innerPath)
	if innerPath != "" {
		innerPath = innerPath + "/"
	}

	files := make([]FileEntry, 0)

	for _, file := range szReader.File {
		normalizedName := utils.NormalizePath(file.Name)

		// Filter by inner path
		if innerPath != "" {
			if innerPath == "/" {
				if strings.Contains(strings.TrimSuffix(normalizedName, "/"), "/") {
					continue
				}
			} else {
				if !strings.HasPrefix(normalizedName, innerPath) {
					continue
				}
				relativePath := strings.TrimPrefix(normalizedName, innerPath)
				if strings.Contains(strings.TrimSuffix(relativePath, "/"), "/") {
					continue
				}
			}
		}

		files = append(files, FileEntry{
			Path:           file.Name,
			Size:           int64(file.UncompressedSize),
			CompressedSize: 0,
			ModTime:        file.Modified,
			IsDir:          file.FileInfo().IsDir(),
		})
	}

	return files, nil
}

// ExtractFile extracts a single file from the 7z archive
func (s *SevenZipFormat) ExtractFile(ctx context.Context, reader io.ReaderAt, size int64, filePath string, password string) (io.ReadCloser, int64, error) {
	var szReader *sevenzip.Reader
	var err error

	if password != "" {
		szReader, err = sevenzip.NewReaderWithPassword(reader, size, password)
	} else {
		szReader, err = sevenzip.NewReader(reader, size)
	}

	if err != nil {
		if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
			if password != "" {
				return nil, 0, ErrPasswordIncorrect
			}
			return nil, 0, ErrPasswordRequired
		}
		return nil, 0, utils.WrapError(err, "failed to open 7z archive")
	}

	filePath = utils.NormalizePath(filePath)

	for _, file := range szReader.File {
		if utils.NormalizePath(file.Name) == filePath {
			rc, err := file.Open()
			if err != nil {
				if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
					if password != "" {
						return nil, 0, ErrPasswordIncorrect
					}
					return nil, 0, ErrPasswordRequired
				}
				return nil, 0, utils.WrapError(err, "failed to open file")
			}

			return rc, int64(file.UncompressedSize), nil
		}
	}

	return nil, 0, ErrFileNotFound
}

func init() {
	RegisterFormat(NewSevenZipFormat())
}
