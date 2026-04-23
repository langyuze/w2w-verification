package handler

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"w2w-verification/internal/store"
)

//go:embed verify.html
var verifyPageHTML []byte

type Handler struct {
	store   *store.Store
	baseURL string
}

func NewHandler(s *store.Store, baseURL string) *Handler {
	return &Handler{store: s, baseURL: baseURL}
}

type storeResponse struct {
	RequestID string `json:"requestId"`
	URL       string `json:"url"`
}

// VerifyHandler handles GET /verify?request={blob}
// Stores the blob and returns a JSON response with the generated UUID and retrieval URL.
func (h *Handler) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqParam := r.URL.Query().Get("request")
	if reqParam == "" {
		http.Error(w, "missing required query parameter: request", http.StatusBadRequest)
		return
	}

	id, err := h.store.Insert(r.Context(), []byte(reqParam))
	if err != nil {
		slog.Error("failed to store data", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storeResponse{
		RequestID: id,
		URL:       h.baseURL + "/getVerificationRequest?requestId=" + id,
	})
}

// GetVerificationRequestHandler handles GET /getVerificationRequest?requestId={uuid}
// Browser requests (Accept: text/html) receive the decryption UI page.
// API requests receive the raw stored blob.
func (h *Handler) GetVerificationRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve the HTML page for browser navigation.
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(verifyPageHTML)
		return
	}

	idParam := r.URL.Query().Get("requestId")
	if idParam == "" {
		http.Error(w, "missing required query parameter: requestId", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(idParam); err != nil {
		http.Error(w, "invalid requestId: must be a valid UUID", http.StatusBadRequest)
		return
	}

	rec, err := h.store.Get(r.Context(), idParam)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to retrieve data", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(rec.Data)
}

// SetVerificationResponseHandler handles POST /setVerificationResponse?requestId={uuid}
// Stores the credential response payload for the given request ID.
func (h *Handler) SetVerificationResponseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("requestId")
	if idParam == "" {
		http.Error(w, "missing required query parameter: requestId", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(idParam); err != nil {
		http.Error(w, "invalid requestId: must be a valid UUID", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if err := h.store.SetResponse(r.Context(), idParam, body); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to store response", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetVerificationResponseHandler handles GET /getVerificationResponse?requestId={uuid}
// Returns the stored credential response, or empty string if none exists.
func (h *Handler) GetVerificationResponseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("requestId")
	if idParam == "" {
		http.Error(w, "missing required query parameter: requestId", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(idParam); err != nil {
		http.Error(w, "invalid requestId: must be a valid UUID", http.StatusBadRequest)
		return
	}

	response, err := h.store.GetResponse(r.Context(), idParam)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("failed to retrieve response", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	if response == nil {
		w.Write([]byte(""))
		return
	}
	w.Write(response)
}
