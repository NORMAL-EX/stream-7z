package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/NORMAL-EX/stream-7z/lib"
	"github.com/NORMAL-EX/stream-7z/lib/formats"
	"go.uber.org/zap"
)

// Request structures for POST JSON APIs
type InfoRequest struct {
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
}

type ListRequest struct {
	URL       string `json:"url"`
	Password  string `json:"password,omitempty"`
	InnerPath string `json:"innerPath,omitempty"`
}

type ExtractRequest struct {
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
	File     string `json:"file"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// InfoResponse represents the response for /api/info
type InfoResponse struct {
	IsEncrypted      bool        `json:"isEncrypted"`
	RequiresPassword bool        `json:"requiresPassword"`
	TotalFiles       int         `json:"totalFiles"`
	TotalSize        int64       `json:"totalSize"`
	Format           string      `json:"format"`
	Comment          string      `json:"comment,omitempty"`
}

// ListResponse represents the response for /api/list
type ListResponse struct {
	Files []FileEntryResponse `json:"files"`
}

// FileEntryResponse represents a file entry in the response
type FileEntryResponse struct {
	Path           string    `json:"path"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressedSize"`
	ModTime        time.Time `json:"modTime"`
	IsDir          bool      `json:"isDir"`
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message, code string) {
	respondJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// parseJSONRequest parses a JSON request body into the provided struct
func parseJSONRequest(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed, use POST", "METHOD_NOT_ALLOWED")
		return &HandlerError{
			Message: "Method not allowed",
			Code:    "METHOD_NOT_ALLOWED",
			Status:  http.StatusMethodNotAllowed,
		}
	}

	if r.Header.Get("Content-Type") != "application/json" {
		respondError(w, http.StatusBadRequest, "Content-Type must be application/json", "INVALID_CONTENT_TYPE")
		return &HandlerError{
			Message: "Invalid content type",
			Code:    "INVALID_CONTENT_TYPE",
			Status:  http.StatusBadRequest,
		}
	}

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error(), "INVALID_JSON")
		return &HandlerError{
			Message: "Invalid JSON",
			Code:    "INVALID_JSON",
			Status:  http.StatusBadRequest,
		}
	}

	return nil
}

// HandlerError represents a handler error
type HandlerError struct {
	Message string
	Code    string
	Status  int
}

func (e *HandlerError) Error() string {
	return e.Message
}

// convertFileEntries converts library file entries to response format
func convertFileEntries(entries []formats.FileEntry) []FileEntryResponse {
	result := make([]FileEntryResponse, len(entries))
	for i, entry := range entries {
		result[i] = FileEntryResponse{
			Path:           entry.Path,
			Size:           entry.Size,
			CompressedSize: entry.CompressedSize,
			ModTime:        entry.ModTime,
			IsDir:          entry.IsDir,
		}
	}
	return result
}

// Handler provides the main HTTP handlers
type Handler struct {
	config *lib.Config
	logger *zap.Logger
}

// NewHandler creates a new Handler instance
func NewHandler(config *lib.Config, logger *zap.Logger) *Handler {
	return &Handler{
		config: config,
		logger: logger,
	}
}

// Health returns a simple health check handler
func (h *Handler) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	}
}
