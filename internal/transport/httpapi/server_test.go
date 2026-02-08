package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"penny-assesment/internal/auth"
)

func TestIssueToken(t *testing.T) {
	authenticator := auth.New("secret", time.Hour)
	handler := NewServer(nil, authenticator)

	payload := map[string]string{"name": "alice", "role": "enduser"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] == "" {
		t.Fatalf("expected token")
	}
}

func TestIssueTokenInvalidRole(t *testing.T) {
	authenticator := auth.New("secret", time.Hour)
	handler := NewServer(nil, authenticator)

	payload := map[string]string{"name": "alice", "role": "invalid"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
}

