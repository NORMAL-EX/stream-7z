package formats

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"io"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib/utils"
	"github.com/ulikunitz/xz"
)

// TarFormat handles TAR archives (including tar.gz, tar.bz2, tar.xz)
type TarFormat struct{}

// NewTarFormat creates a new TAR format handler
func NewTarFormat() *TarFormat {
	return &TarFormat{}
}

// Name returns the format name
func (t *TarFormat) Name() string {
	return "tar"
}

// Extensions returns supported file extensions
func (t *TarFormat) Extensions() []string {
	return []string{".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz"}
}

// Detect checks if the reader contains a TAR archive
func (t *TarFormat) Detect(ctx context.Context, reader io.ReaderAt, size int64) (bool, error) {
	// Check various compression formats
	magic := make([]byte, 10)
	if _, err := reader.ReadAt(magic, 0); err != nil {
		return false, err
	}

	// Gzip: 0x1f 0x8b
	if magic[0] == 0x1f && magic[1] == 0x8b {
		return true, nil
	}

	// Bzip2: BZ
	if magic[0] == 'B' && magic[1] == 'Z' {
		return true, nil
	}

	// XZ: 0xFD 0x37 0x7A 0x58 0x5A 0x00
	if magic[0] == 0xFD && magic[1] == 0x37 && magic[2] == 0x7A &&
		magic[3] == 0x58 && magic[4] == 0x5A && magic[5] == 0x00 {
		return true, nil
	}

	// Plain TAR: Check for "ustar" at offset 257
	ustar := make([]byte, 6)
	if _, err := reader.ReadAt(ustar, 257); err == nil {
		if string(ustar[:5]) == "ustar" {
			return true, nil
		}
	}

	return false, nil
}

// detectCompression determines the compression type
func (t *TarFormat) detectCompression(reader io.ReaderAt) (string, error) {
	magic := make([]byte, 10)
	if _, err := reader.ReadAt(magic, 0); err != nil {
		return "", err
	}

	// Gzip
	if magic[0] == 0x1f && magic[1] == 0x8b {
		return "gzip", nil
	}

	// Bzip2
	if magic[0] == 'B' && magic[1] == 'Z' {
		return "bzip2", nil
	}

	// XZ
	if magic[0] == 0xFD && magic[1] == 0x37 && magic[2] == 0x7A &&
		magic[3] == 0x58 && magic[4] == 0x5A && magic[5] == 0x00 {
		return "xz", nil
	}

	return "none", nil
}

// wrapReader wraps the reader with appropriate decompression
func (t *TarFormat) wrapReader(reader io.Reader, compression string) (io.Reader, error) {
	switch compression {
	case "gzip":
		return gzip.NewReader(reader)
	case "bzip2":
		return bzip2.NewReader(reader), nil
	case "xz":
		return xz.NewReader(reader)
	case "none":
		return reader, nil
	default:
		return reader, nil
	}
}

// GetInfo retrieves metadata about the TAR archive
func (t *TarFormat) GetInfo(ctx context.Context, reader io.ReaderAt, size int64, password string) (*ArchiveInfo, error) {
	// TAR doesn't support encryption
	if password != "" {
		return nil, &FormatError{Message: "TAR format does not support encryption"}
	}

	compression, err := t.detectCompression(reader)
	if err != nil {
		return nil, err
	}

	sectionReader := io.NewSectionReader(reader, 0, size)
	wrappedReader, err := t.wrapReader(sectionReader, compression)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create decompressor")
	}

	tarReader := tar.NewReader(wrappedReader)

	info := &ArchiveInfo{
		IsEncrypted:      false,
		RequiresPassword: false,
		TotalFiles:       0,
		TotalSize:        0,
		Files:            make([]FileEntry, 0),
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, utils.WrapError(err, "failed to read TAR header")
		}

		entry := FileEntry{
			Path:           header.Name,
			Size:           header.Size,
			CompressedSize: 0, // TAR doesn't store individual compressed sizes
			ModTime:        header.ModTime,
			IsDir:          header.Typeflag == tar.TypeDir,
		}

		info.Files = append(info.Files, entry)

		if !entry.IsDir {
			info.TotalFiles++
			info.TotalSize += entry.Size
		}
	}

	return info, nil
}

// ListFiles returns a list of files in the TAR archive
func (t *TarFormat) ListFiles(ctx context.Context, reader io.ReaderAt, size int64, innerPath string, password string) ([]FileEntry, error) {
	if password != "" {
		return nil, &FormatError{Message: "TAR format does not support encryption"}
	}

	compression, err := t.detectCompression(reader)
	if err != nil {
		return nil, err
	}

	sectionReader := io.NewSectionReader(reader, 0, size)
	wrappedReader, err := t.wrapReader(sectionReader, compression)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create decompressor")
	}

	tarReader := tar.NewReader(wrappedReader)

	innerPath = utils.NormalizePath(innerPath)
	if innerPath != "" {
		innerPath = innerPath + "/"
	}

	files := make([]FileEntry, 0)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, utils.WrapError(err, "failed to read TAR header")
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
			Size:           header.Size,
			CompressedSize: 0,
			ModTime:        header.ModTime,
			IsDir:          header.Typeflag == tar.TypeDir,
		})
	}

	return files, nil
}

// ExtractFile extracts a single file from the TAR archive
func (t *TarFormat) ExtractFile(ctx context.Context, reader io.ReaderAt, size int64, filePath string, password string) (io.ReadCloser, int64, error) {
	if password != "" {
		return nil, 0, &FormatError{Message: "TAR format does not support encryption"}
	}

	compression, err := t.detectCompression(reader)
	if err != nil {
		return nil, 0, err
	}

	sectionReader := io.NewSectionReader(reader, 0, size)
	wrappedReader, err := t.wrapReader(sectionReader, compression)
	if err != nil {
		return nil, 0, utils.WrapError(err, "failed to create decompressor")
	}

	tarReader := tar.NewReader(wrappedReader)
	filePath = utils.NormalizePath(filePath)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, utils.WrapError(err, "failed to read TAR header")
		}

		if utils.NormalizePath(header.Name) == filePath {
			// TAR reader doesn't support seeking, so we return it as-is
			// The caller must read it immediately
			return io.NopCloser(tarReader), header.Size, nil
		}
	}

	return nil, 0, ErrFileNotFound
}

func init() {
	RegisterFormat(NewTarFormat())
}
