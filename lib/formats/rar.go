package formats

import (
	"context"
	"io"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib/utils"
	"github.com/nwaples/rardecode/v2"
)

// RarFormat handles RAR archives (RAR4 and RAR5)
type RarFormat struct{}

// NewRarFormat creates a new RAR format handler
func NewRarFormat() *RarFormat {
	return &RarFormat{}
}

// Name returns the format name
func (r *RarFormat) Name() string {
	return "rar"
}

// Extensions returns supported file extensions
func (r *RarFormat) Extensions() []string {
	return []string{".rar"}
}

// Detect checks if the reader contains a RAR archive
func (r *RarFormat) Detect(ctx context.Context, reader io.ReaderAt, size int64) (bool, error) {
	// Check RAR magic numbers
	magic := make([]byte, 8)
	if _, err := reader.ReadAt(magic, 0); err != nil {
		return false, err
	}

	// RAR 4.x: Rar!\x1a\x07\x00
	if len(magic) >= 7 &&
		magic[0] == 'R' && magic[1] == 'a' && magic[2] == 'r' && magic[3] == '!' &&
		magic[4] == 0x1a && magic[5] == 0x07 && magic[6] == 0x00 {
		return true, nil
	}

	// RAR 5.x: Rar!\x1a\x07\x01\x00
	if len(magic) >= 8 &&
		magic[0] == 'R' && magic[1] == 'a' && magic[2] == 'r' && magic[3] == '!' &&
		magic[4] == 0x1a && magic[5] == 0x07 && magic[6] == 0x01 && magic[7] == 0x00 {
		return true, nil
	}

	return false, nil
}

// GetInfo retrieves metadata about the RAR archive
func (r *RarFormat) GetInfo(ctx context.Context, reader io.ReaderAt, size int64, password string) (*ArchiveInfo, error) {
	sectionReader := io.NewSectionReader(reader, 0, size)

	var rarReader *rardecode.Reader
	var err error

	if password != "" {
		rarReader, err = rardecode.NewReader(sectionReader, rardecode.Password(password))
	} else {
		rarReader, err = rardecode.NewReader(sectionReader)
	}

	if err != nil {
		return nil, utils.WrapError(err, "failed to open RAR archive")
	}

	info := &ArchiveInfo{
		IsEncrypted:      false,
		RequiresPassword: false,
		TotalFiles:       0,
		TotalSize:        0,
		Files:            make([]FileEntry, 0),
	}

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Check if error is due to encryption
			if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
				info.IsEncrypted = true
				info.RequiresPassword = true
				if password != "" {
					return info, ErrPasswordIncorrect
				}
				return info, ErrPasswordRequired
			}
			return nil, utils.WrapError(err, "failed to read RAR header")
		}

		// Check if this file is encrypted (using HostOS as a proxy check since IsEncrypted field doesn't exist)
		// RAR encryption is detected via errors, so we rely on the error handling above
		
		entry := FileEntry{
			Path:           header.Name,
			Size:           header.UnPackedSize,
			CompressedSize: header.PackedSize,
			ModTime:        header.ModificationTime,
			IsDir:          header.IsDir,
		}

		info.Files = append(info.Files, entry)

		if !header.IsDir {
			info.TotalFiles++
			info.TotalSize += entry.Size
		}
	}

	if info.IsEncrypted && password == "" {
		info.RequiresPassword = true
		return info, ErrPasswordRequired
	}

	return info, nil
}

// ListFiles returns a list of files in the RAR archive
func (r *RarFormat) ListFiles(ctx context.Context, reader io.ReaderAt, size int64, innerPath string, password string) ([]FileEntry, error) {
	sectionReader := io.NewSectionReader(reader, 0, size)

	var rarReader *rardecode.Reader
	var err error

	if password != "" {
		rarReader, err = rardecode.NewReader(sectionReader, rardecode.Password(password))
	} else {
		rarReader, err = rardecode.NewReader(sectionReader)
	}

	if err != nil {
		return nil, utils.WrapError(err, "failed to open RAR archive")
	}

	innerPath = utils.NormalizePath(innerPath)
	if innerPath != "" {
		innerPath = innerPath + "/"
	}

	files := make([]FileEntry, 0)

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
				if password != "" {
					return nil, ErrPasswordIncorrect
				}
				return nil, ErrPasswordRequired
			}
			return nil, utils.WrapError(err, "failed to read RAR header")
		}

		normalizedName := utils.NormalizePath(header.Name)

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
			Path:           header.Name,
			Size:           header.UnPackedSize,
			CompressedSize: header.PackedSize,
			ModTime:        header.ModificationTime,
			IsDir:          header.IsDir,
		})
	}

	return files, nil
}

// ExtractFile extracts a single file from the RAR archive
func (r *RarFormat) ExtractFile(ctx context.Context, reader io.ReaderAt, size int64, filePath string, password string) (io.ReadCloser, int64, error) {
	sectionReader := io.NewSectionReader(reader, 0, size)

	var rarReader *rardecode.Reader
	var err error

	if password != "" {
		rarReader, err = rardecode.NewReader(sectionReader, rardecode.Password(password))
	} else {
		rarReader, err = rardecode.NewReader(sectionReader)
	}

	if err != nil {
		return nil, 0, utils.WrapError(err, "failed to open RAR archive")
	}

	filePath = utils.NormalizePath(filePath)

	for {
		header, err := rarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "encrypted") {
				if password != "" {
					return nil, 0, ErrPasswordIncorrect
				}
				return nil, 0, ErrPasswordRequired
			}
			return nil, 0, utils.WrapError(err, "failed to read RAR header")
		}

		if utils.NormalizePath(header.Name) == filePath {
			// RAR reader doesn't support seeking, so we need to read the entire file
			// into memory or a temp file
			return io.NopCloser(rarReader), header.UnPackedSize, nil
		}
	}

	return nil, 0, ErrFileNotFound
}

func init() {
	RegisterFormat(NewRarFormat())
}
