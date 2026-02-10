package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MEKXH/golem/internal/version"
)

type mockChatProcessor struct {
	gotSession string
	gotSender  string
	gotMessage string
	resp       string
	err        error
}

func (m *mockChatProcessor) ProcessForChannel(ctx context.Context, channel, chatID, senderID, content string) (string, error) {
	m.gotSession = channel + ":" + chatID
	m.gotSender = senderID
	m.gotMessage = content
	if m.err != nil {
		return "", m.err
	}
	return m.resp, nil
}

func decodeJSON(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return out
}

func TestHealthEndpoint(t *testing.T) {
	h := NewHandler("", &mockChatProcessor{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := decodeJSON(t, rr.Body)
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", body["status"])
	}
	if body["request_id"] == "" {
		t.Fatal("expected non-empty request_id")
	}
}

func TestVersionEndpoint(t *testing.T) {
	h := NewHandler("", &mockChatProcessor{})
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := decodeJSON(t, rr.Body)
	if body["version"] != version.Version {
		t.Fatalf("expected version=%s, got %v", version.Version, body["version"])
	}
}

func TestChatUnauthorized(t *testing.T) {
	h := NewHandler("secret-token", &mockChatProcessor{resp: "ok"})
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	body := decodeJSON(t, rr.Body)
	if body["code"] != "unauthorized" {
		t.Fatalf("expected code=unauthorized, got %v", body["code"])
	}
}

func TestChatBadRequest(t *testing.T) {
	h := NewHandler("", &mockChatProcessor{})
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	body := decodeJSON(t, rr.Body)
	if body["code"] != "bad_request" {
		t.Fatalf("expected code=bad_request, got %v", body["code"])
	}
}

func TestChatSuccess(t *testing.T) {
	processor := &mockChatProcessor{resp: "hello back"}
	h := NewHandler("secret-token", processor)
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello","session_id":"s1","sender_id":"u1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if processor.gotSession != "gateway:s1" {
		t.Fatalf("expected session gateway:s1, got %s", processor.gotSession)
	}
	if processor.gotSender != "u1" {
		t.Fatalf("expected sender u1, got %s", processor.gotSender)
	}
	if processor.gotMessage != "hello" {
		t.Fatalf("expected message hello, got %s", processor.gotMessage)
	}

	body := decodeJSON(t, rr.Body)
	if body["response"] != "hello back" {
		t.Fatalf("expected response=hello back, got %v", body["response"])
	}
	if body["session_id"] != "s1" {
		t.Fatalf("expected session_id=s1, got %v", body["session_id"])
	}
}

func TestChatInternalError(t *testing.T) {
	processor := &mockChatProcessor{err: errors.New("model down")}
	h := NewHandler("", processor)
	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
	body := decodeJSON(t, rr.Body)
	if body["code"] != "internal_error" {
		t.Fatalf("expected code=internal_error, got %v", body["code"])
	}
}
