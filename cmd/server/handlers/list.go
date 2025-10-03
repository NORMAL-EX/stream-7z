package handlers

import (
	"net/http"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib"
	"go.uber.org/zap"
)

// List handles POST /api/list requests
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse JSON request
		var req ListRequest
		if err := parseJSONRequest(w, r, &req); err != nil {
			return
		}

		// Validate URL
		if req.URL == "" {
			respondError(w, http.StatusBadRequest, "url is required", "MISSING_URL")
			return
		}

		h.logger.Info("listing archive files",
			zap.String("url", req.URL),
			zap.String("inner_path", req.InnerPath),
			zap.Bool("has_password", req.Password != ""),
		)

		// List files using QuickList
		files, err := lib.QuickList(req.URL, req.InnerPath, req.Password, h.config)
		if err != nil {
			h.logger.Error("failed to list archive files",
				zap.String("url", req.URL),
				zap.String("inner_path", req.InnerPath),
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
				respondError(w, http.StatusNotFound, "Path not found in archive", "PATH_NOT_FOUND")
			} else if strings.Contains(errMsg, "unsupported") || strings.Contains(errMsg, "format") {
				respondError(w, http.StatusBadRequest, "Unsupported archive format", "UNSUPPORTED_FORMAT")
			} else if strings.Contains(errMsg, "URL") || strings.Contains(errMsg, "request failed") {
				respondError(w, http.StatusBadRequest, "Failed to access URL", "URL_ERROR")
			} else {
				respondError(w, http.StatusInternalServerError, "Failed to list files", "INTERNAL_ERROR")
			}
			return
		}

		// Convert to response format
		response := ListResponse{
			Files: convertFileEntries(files),
		}

		h.logger.Info("successfully listed archive files",
			zap.String("url", req.URL),
			zap.String("inner_path", req.InnerPath),
			zap.Int("file_count", len(files)),
		)

		respondJSON(w, http.StatusOK, response)
	}
}
