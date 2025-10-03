package formats

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib/utils"
	"github.com/saintfish/chardet"
	"github.com/yeka/zip"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// ZipFormat handles ZIP archives
type ZipFormat struct{}

// NewZipFormat creates a new ZIP format handler
func NewZipFormat() *ZipFormat {
	return &ZipFormat{}
}

// Name returns the format name
func (z *ZipFormat) Name() string {
	return "zip"
}

// Extensions returns supported file extensions
func (z *ZipFormat) Extensions() []string {
	return []string{".zip"}
}

// Detect checks if the reader contains a ZIP archive
func (z *ZipFormat) Detect(ctx context.Context, reader io.ReaderAt, size int64) (bool, error) {
	// Check ZIP magic number
	magic := make([]byte, 4)
	if _, err := reader.ReadAt(magic, 0); err != nil {
		return false, err
	}

	// ZIP files start with PK\x03\x04 or PK\x05\x06 (empty archive) or PK\x07\x08 (spanned archive)
	if magic[0] == 'P' && magic[1] == 'K' &&
		((magic[2] == 0x03 && magic[3] == 0x04) ||
			(magic[2] == 0x05 && magic[3] == 0x06) ||
			(magic[2] == 0x07 && magic[3] == 0x08)) {
		return true, nil
	}

	return false, nil
}

// GetInfo retrieves metadata about the ZIP archive
func (z *ZipFormat) GetInfo(ctx context.Context, reader io.ReaderAt, size int64, password string) (*ArchiveInfo, error) {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, utils.WrapError(err, "failed to open ZIP archive")
	}

	info := &ArchiveInfo{
		IsEncrypted:      false,
		RequiresPassword: false,
		TotalFiles:       0,
		TotalSize:        0,
		Files:            make([]FileEntry, 0),
		Comment:          zipReader.Comment,
	}

	passwordVerified := false

	for _, file := range zipReader.File {
		fileName := decodeName(file.Name)

		// Check if file is encrypted
		if file.IsEncrypted() {
			info.IsEncrypted = true
			if !passwordVerified && password != "" {
				// Verify password by trying to open the file
				file.SetPassword(password)
				rc, err := file.Open()
				if err != nil {
					if strings.Contains(err.Error(), "password") {
						info.RequiresPassword = true
						return info, ErrPasswordIncorrect
					}
					return nil, utils.WrapError(err, "failed to verify password")
				}
				rc.Close()
				passwordVerified = true
			} else if password == "" {
				info.RequiresPassword = true
			}
		}

		isDir := strings.HasSuffix(fileName, "/") || file.FileInfo().IsDir()

		entry := FileEntry{
			Path:           fileName,
			Size:           int64(file.UncompressedSize64),
			CompressedSize: int64(file.CompressedSize64),
			ModTime:        file.FileInfo().ModTime(),
			IsDir:          isDir,
		}

		info.Files = append(info.Files, entry)

		if !isDir {
			info.TotalFiles++
			info.TotalSize += entry.Size
		}
	}

	// If archive is encrypted and password wasn't provided, indicate it's required
	if info.IsEncrypted && password == "" {
		info.RequiresPassword = true
		return info, ErrPasswordRequired
	}

	return info, nil
}

// ListFiles returns a list of files in the ZIP archive
func (z *ZipFormat) ListFiles(ctx context.Context, reader io.ReaderAt, size int64, innerPath string, password string) ([]FileEntry, error) {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, utils.WrapError(err, "failed to open ZIP archive")
	}

	// 关键修复: 在 NormalizePath 之前保存原始输入
	// 空字符串 "" 表示列出所有文件（递归）
	// "/" 表示列出根目录的直接子项（非递归）
	originalInnerPath := innerPath
	listAll := (originalInnerPath == "")
	isRoot := (originalInnerPath == "/")
	
	innerPath = utils.NormalizePath(innerPath)
	
	// NormalizePath 会把 "" 和 "/" 都转换为 "."
	// 需要根据原始输入来决定最终值
	if innerPath == "." {
		if listAll {
			innerPath = ""
		} else if isRoot {
			innerPath = ""
		} else {
			innerPath = ""
		}
	}
	
	// 只有在指定了具体目录时才添加后缀 "/"
	if innerPath != "" && !listAll && !isRoot {
		innerPath = innerPath + "/"
	}

	files := make([]FileEntry, 0)
	passwordVerified := false

	for _, file := range zipReader.File {
		fileName := decodeName(file.Name)
		normalizedName := utils.NormalizePath(fileName)

		// 根据不同情况进行过滤
		if listAll {
			// 列出所有文件，不做任何过滤
		} else if isRoot {
			// 根目录 - 只列出第一层的文件和目录
			// normalizedName 已经移除了前导 "/"
			// 例如: "file.txt" 或 "dir/file.txt"
			// 我们只要第一层，即不包含 "/" 的，或者以 "/" 结尾的目录
			if strings.Contains(strings.TrimSuffix(normalizedName, "/"), "/") {
				continue
			}
		} else {
			// 指定目录 - 只列出该目录下的直接子项
			if !strings.HasPrefix(normalizedName, innerPath) {
				continue
			}
			// Remove the inner path prefix
			relativePath := strings.TrimPrefix(normalizedName, innerPath)
			// Only include direct children
			if strings.Contains(strings.TrimSuffix(relativePath, "/"), "/") {
				continue
			}
		}

		// Verify password if file is encrypted
		if file.IsEncrypted() {
			if !passwordVerified {
				if password == "" {
					return nil, ErrPasswordRequired
				}
				file.SetPassword(password)
				rc, err := file.Open()
				if err != nil {
					if strings.Contains(err.Error(), "password") {
						return nil, ErrPasswordIncorrect
					}
					return nil, utils.WrapError(err, "failed to open encrypted file")
				}
				rc.Close()
				passwordVerified = true
			} else {
				file.SetPassword(password)
			}
		}

		isDir := strings.HasSuffix(fileName, "/") || file.FileInfo().IsDir()

		files = append(files, FileEntry{
			Path:           fileName,
			Size:           int64(file.UncompressedSize64),
			CompressedSize: int64(file.CompressedSize64),
			ModTime:        file.FileInfo().ModTime(),
			IsDir:          isDir,
		})
	}

	return files, nil
}

// ExtractFile extracts a single file from the ZIP archive
func (z *ZipFormat) ExtractFile(ctx context.Context, reader io.ReaderAt, size int64, filePath string, password string) (io.ReadCloser, int64, error) {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, 0, utils.WrapError(err, "failed to open ZIP archive")
	}

	filePath = utils.NormalizePath(filePath)

	for _, file := range zipReader.File {
		fileName := decodeName(file.Name)
		if utils.NormalizePath(fileName) == filePath {
			// Set password if file is encrypted
			if file.IsEncrypted() {
				if password == "" {
					return nil, 0, ErrPasswordRequired
				}
				file.SetPassword(password)
			}

			rc, err := file.Open()
			if err != nil {
				if file.IsEncrypted() && strings.Contains(err.Error(), "password") {
					return nil, 0, ErrPasswordIncorrect
				}
				return nil, 0, utils.WrapError(err, "failed to open file")
			}

			return rc, int64(file.UncompressedSize64), nil
		}
	}

	return nil, 0, ErrFileNotFound
}

// decodeName handles various character encodings in ZIP file names
func decodeName(name string) string {
	b := []byte(name)
	detector := chardet.NewTextDetector()
	results, err := detector.DetectAll(b)
	if err != nil {
		return name
	}

	var enc encoding.Encoding
	for _, r := range results {
		if r.Confidence > 30 {
			enc = getEncoding(r.Charset)
			if enc != nil {
				break
			}
		}
	}

	if enc == nil {
		return name
	}

	decoder := transform.NewReader(bytes.NewReader(b), enc.NewDecoder())
	content, err := io.ReadAll(decoder)
	if err != nil {
		return name
	}

	return string(content)
}

// getEncoding returns the appropriate encoding for the given charset name
func getEncoding(charset string) encoding.Encoding {
	switch charset {
	case "UTF-8":
		return unicode.UTF8
	case "UTF-16BE":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	case "UTF-16LE":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "ISO-8859-1":
		return charmap.ISO8859_1
	case "ISO-8859-2":
		return charmap.ISO8859_2
	case "windows-1251":
		return charmap.Windows1251
	case "Shift_JIS":
		return japanese.ShiftJIS
	case "GB-18030", "GB18030":
		return simplifiedchinese.GB18030
	case "GBK":
		return simplifiedchinese.GBK
	case "EUC-KR":
		return korean.EUCKR
	case "Big5":
		return traditionalchinese.Big5
	default:
		return nil
	}
}

func init() {
	RegisterFormat(NewZipFormat())
}
