package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib"
	"go.uber.org/zap"
)

// Extract handles POST /api/extract requests
func (h *Handler) Extract() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse JSON request
		var req ExtractRequest
		if err := parseJSONRequest(w, r, &req); err != nil {
			return
		}

		// Validate URL and file path
		if req.URL == "" {
			respondError(w, http.StatusBadRequest, "url is required", "MISSING_URL")
			return
		}

		if req.File == "" {
			respondError(w, http.StatusBadRequest, "file is required", "MISSING_FILE")
			return
		}

		h.logger.Info("extracting file from archive",
			zap.String("url", req.URL),
			zap.String("file_path", req.File),
			zap.Bool("has_password", req.Password != ""),
		)

		// Extract file using QuickExtract
		reader, size, err := lib.QuickExtract(req.URL, req.File, req.Password, h.config)
		if err != nil {
			h.logger.Error("failed to extract file",
				zap.String("url", req.URL),
				zap.String("file_path", req.File),
				zap.Error(err),
			)

			// Determine error type
			errMsg := err.Error()
			if strings.Contains(errMsg, "password") {
				if req.Password != "" {
					respondError(w, http.StatusUnauthorized, "Incorrect password", "WRONG_PASSWORD")
				} else {
					respondError(w, http.StatusUnauthorized, "Password required", "PASSWORD_REQUIRED")
				}
			} else if strings.Contains(errMsg, "not found") {
				respondError(w, http.StatusNotFound, "File not found in archive", "FILE_NOT_FOUND")
			} else if strings.Contains(errMsg, "unsupported") || strings.Contains(errMsg, "format") {
				respondError(w, http.StatusBadRequest, "Unsupported archive format", "UNSUPPORTED_FORMAT")
			} else if strings.Contains(errMsg, "URL") || strings.Contains(errMsg, "request failed") {
				respondError(w, http.StatusBadRequest, "Failed to access URL", "URL_ERROR")
			} else if strings.Contains(errMsg, "path traversal") {
				respondError(w, http.StatusBadRequest, "Invalid file path", "INVALID_PATH")
			} else {
				respondError(w, http.StatusInternalServerError, "Failed to extract file", "INTERNAL_ERROR")
			}
			return
		}
		defer reader.Close()

		// Get filename from path
		filename := filepath.Base(req.File)

		// Set headers for file download
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

		// Stream file to response
		written, err := io.Copy(w, reader)
		if err != nil {
			h.logger.Error("failed to stream file",
				zap.String("url", req.URL),
				zap.String("file_path", req.File),
				zap.Int64("written", written),
				zap.Error(err),
			)
			return
		}

		h.logger.Info("successfully extracted file",
			zap.String("url", req.URL),
			zap.String("file_path", req.File),
			zap.Int64("size", size),
			zap.Int64("written", written),
		)
	}
}
