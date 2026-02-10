package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSearch_NoAPIKey(t *testing.T) {
	impl := &webSearchToolImpl{
		apiKey:     "",
		maxResults: 5,
		endpoint:   "https://example.com/search",
		client:     &http.Client{Timeout: 5 * time.Second},
	}

	_, err := impl.execute(context.Background(), &WebSearchInput{Query: "golem"})
	if err == nil {
		t.Fatal("expected error when API key is missing")
	}
}

func TestWebSearch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			t.Fatalf("missing/invalid brave API header")
		}
		if got := r.URL.Query().Get("q"); got != "golem" {
			t.Fatalf("unexpected query: %s", got)
		}
		if got := r.URL.Query().Get("count"); got != "3" {
			t.Fatalf("unexpected count: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "web": {
    "results": [
      {"title":"Golem", "url":"https://example.com", "description":"AI assistant"}
    ]
  }
}`))
	}))
	defer server.Close()

	impl := &webSearchToolImpl{
		apiKey:     "test-key",
		maxResults: 5,
		endpoint:   server.URL,
		client:     server.Client(),
	}

	out, err := impl.execute(context.Background(), &WebSearchInput{
		Query:      "golem",
		MaxResults: 3,
	})
	if err != nil {
		t.Fatalf("web search error: %v", err)
	}
	if out.Query != "golem" {
		t.Fatalf("unexpected query in output: %s", out.Query)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	if out.Results[0].URL != "https://example.com" {
		t.Fatalf("unexpected url: %s", out.Results[0].URL)
	}
}

func TestWebFetch_HTMLToText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>Title</h1><p>Hello <b>Golem</b></p></body></html>`))
	}))
	defer server.Close()

	impl := &webFetchToolImpl{
		client:   server.Client(),
		maxBytes: 1024,
	}

	out, err := impl.execute(context.Background(), &WebFetchInput{URL: server.URL})
	if err != nil {
		t.Fatalf("web fetch error: %v", err)
	}
	if out.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", out.Status)
	}
	if !strings.Contains(out.Content, "Title") || !strings.Contains(out.Content, "Hello Golem") {
		t.Fatalf("unexpected content: %s", out.Content)
	}
}

func TestWebFetch_TruncatesLargeContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("x", 200)))
	}))
	defer server.Close()

	impl := &webFetchToolImpl{
		client:   server.Client(),
		maxBytes: 64,
	}

	out, err := impl.execute(context.Background(), &WebFetchInput{URL: server.URL})
	if err != nil {
		t.Fatalf("web fetch error: %v", err)
	}
	if !out.Truncated {
		t.Fatal("expected truncated=true for oversized body")
	}
	if len(out.Content) > 64 {
		t.Fatalf("expected content length <= 64, got %d", len(out.Content))
	}
}
