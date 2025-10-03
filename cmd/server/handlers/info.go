package handlers

import (
	"net/http"
	"strings"

	"github.com/NORMAL-EX/stream-7z/lib"
	"go.uber.org/zap"
)

// Info handles POST /api/info requests
func (h *Handler) Info() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse JSON request
		var req InfoRequest
		if err := parseJSONRequest(w, r, &req); err != nil {
			return
		}

		// Validate URL
		if req.URL == "" {
			respondError(w, http.StatusBadRequest, "url is required", "MISSING_URL")
			return
		}

		h.logger.Info("getting archive info",
			zap.String("url", req.URL),
			zap.Bool("has_password", req.Password != ""),
		)

		// Get archive info using QuickInfo
		info, err := lib.QuickInfo(req.URL, req.Password, h.config)
		if err != nil {
			h.logger.Error("failed to get archive info",
				zap.String("url", req.URL),
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
			} else if strings.Contains(errMsg, "unsupported") || strings.Contains(errMsg, "format") {
				respondError(w, http.StatusBadRequest, "Unsupported archive format", "UNSUPPORTED_FORMAT")
			} else if strings.Contains(errMsg, "URL") || strings.Contains(errMsg, "request failed") {
				respondError(w, http.StatusBadRequest, "Failed to access URL", "URL_ERROR")
			} else {
				respondError(w, http.StatusInternalServerError, "Failed to get archive info", "INTERNAL_ERROR")
			}
			return
		}

		// Create response
		response := InfoResponse{
			IsEncrypted:      info.IsEncrypted,
			RequiresPassword: info.RequiresPassword,
			TotalFiles:       info.TotalFiles,
			TotalSize:        info.TotalSize,
			Comment:          info.Comment,
		}

		// Get format from a new archive instance (since QuickInfo closed it)
		archive, err := lib.NewArchive(req.URL, h.config)
		if err == nil {
			response.Format = archive.Format()
			archive.Close()
		}

		h.logger.Info("successfully retrieved archive info",
			zap.String("url", req.URL),
			zap.Int("total_files", info.TotalFiles),
			zap.Int64("total_size", info.TotalSize),
		)

		respondJSON(w, http.StatusOK, response)
	}
}
