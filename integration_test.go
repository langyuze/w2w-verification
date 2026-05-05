package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	mux.HandleFunc("/setVerificationResponse", h.SetVerificationResponseHandler)
	mux.HandleFunc("/getVerificationResponse", h.GetVerificationResponseHandler)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func postVerify(t *testing.T, ts *httptest.Server, body string) (string, string) {
	t.Helper()
	resp, err := http.Post(ts.URL+"/verify", "application/octet-stream", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /verify: %v", err)
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
	return result.RequestID, result.URL
}

func TestStoreAndRetrieveRoundTrip(t *testing.T) {
	ts := setupTestServer(t)

	requestID, resultURL := postVerify(t, ts, "hello world")

	if requestID == "" {
		t.Fatal("empty requestId in response")
	}
	expectedURL := "https://demo.verifiedbygoogle.com/getVerificationRequest?requestId=" + requestID
	if resultURL != expectedURL {
		t.Errorf("url mismatch: got %q, want %q", resultURL, expectedURL)
	}

	// Retrieve
	resp2, err := http.Get(ts.URL + "/getVerificationRequest?requestId=" + requestID)
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

func TestEmptyBody(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Post(ts.URL+"/verify", "application/octet-stream", strings.NewReader(""))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestVerifyRejectsGet(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/verify")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
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

	data := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}

	resp, err := http.Post(ts.URL+"/verify", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST /verify: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		RequestID string `json:"requestId"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	resp2, err := http.Get(ts.URL + "/getVerificationRequest?requestId=" + result.RequestID)
	if err != nil {
		t.Fatalf("GET /getVerificationRequest: %v", err)
	}
	defer resp2.Body.Close()

	body, _ := io.ReadAll(resp2.Body)
	if !bytes.Equal(body, data) {
		t.Errorf("binary data mismatch: got %x, want %x", body, data)
	}
}

func TestVerificationResponseRoundTrip(t *testing.T) {
	ts := setupTestServer(t)

	requestID, _ := postVerify(t, ts, "test payload")

	// Response should be empty initially
	resp, err := http.Get(ts.URL + "/getVerificationResponse?requestId=" + requestID)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "" {
		t.Errorf("expected empty response, got %q", body)
	}

	// Set a response
	responsePayload := `{"data":"test-credential","protocol":"openid4vp"}`
	setResp, err := http.Post(ts.URL+"/setVerificationResponse?requestId="+requestID, "application/json", strings.NewReader(responsePayload))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	setResp.Body.Close()
	if setResp.StatusCode != http.StatusOK {
		t.Fatalf("set response status: got %d, want %d", setResp.StatusCode, http.StatusOK)
	}

	// Retrieve the response
	resp2, err := http.Get(ts.URL + "/getVerificationResponse?requestId=" + requestID)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != responsePayload {
		t.Errorf("response mismatch: got %q, want %q", body2, responsePayload)
	}
}
