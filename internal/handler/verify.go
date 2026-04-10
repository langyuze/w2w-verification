package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"w2w-verification/internal/store"
)

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
// Returns the stored blob for the given UUID.
func (h *Handler) GetVerificationRequestHandler(w http.ResponseWriter, r *http.Request) {
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
