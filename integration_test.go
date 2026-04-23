package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"w2w-verification/internal/handler"
	"w2w-verification/internal/store"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := handler.NewHandler(s, "https://demo.verifiedbygoogle.com")

	mux := http.NewServeMux()
	mux.HandleFunc("/verify", h.VerifyHandler)
	mux.HandleFunc("/getVerificationRequest", h.GetVerificationRequestHandler)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestStoreAndRetrieveRoundTrip(t *testing.T) {
	ts := setupTestServer(t)

	// Store
	storeURL := ts.URL + "/verify?request=" + url.QueryEscape("hello world")
	resp, err := http.Get(storeURL)
	if err != nil {
		t.Fatalf("GET /verify: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("store status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result struct {
		RequestID string `json:"requestId"`
		URL       string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.RequestID == "" {
		t.Fatal("empty requestId in response")
	}
	expectedURL := "https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=" + result.RequestID
	if result.URL != expectedURL {
		t.Errorf("url mismatch: got %q, want %q", result.URL, expectedURL)
	}

	// Retrieve
	getURL := ts.URL + "/getVerificationRequest?requestId=" + result.RequestID
	resp2, err := http.Get(getURL)
	if err != nil {
		t.Fatalf("GET /getVerificationRequest: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("retrieve status: got %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp2.Body)
	if string(body) != "hello world" {
		t.Errorf("data mismatch: got %q, want %q", body, "hello world")
	}
}

func TestRetrieveNonExistent(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/getVerificationRequest?requestId=00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestMissingRequestParam(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/verify")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestMissingRequestIdParam(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/getVerificationRequest")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestInvalidUUID(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/getVerificationRequest?requestId=not-a-uuid")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestGetVerificationRequestServesHTMLForBrowser(t *testing.T) {
	ts := setupTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/getVerificationRequest?requestId=00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type: got %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "navigator.credentials.get") {
		t.Error("HTML page missing expected JS content")
	}
}

func TestBinaryDataRoundTrip(t *testing.T) {
	ts := setupTestServer(t)

	// Binary data with null bytes and high bytes
	data := "\x00\x01\x02\xff\xfe\xfd"

	storeURL := ts.URL + "/verify?request=" + url.QueryEscape(data)
	resp, err := http.Get(storeURL)
	if err != nil {
		t.Fatalf("GET /verify: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		RequestID string `json:"requestId"`
		URL       string `json:"url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	resp2, err := http.Get(ts.URL + "/getVerificationRequest?requestId=" + result.RequestID)
	if err != nil {
		t.Fatalf("GET /getVerificationRequest: %v", err)
	}
	defer resp2.Body.Close()

	body, _ := io.ReadAll(resp2.Body)
	if string(body) != data {
		t.Errorf("binary data mismatch: got %x, want %x", body, data)
	}
}
