package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInstallerSearch_ReturnsAvailableSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"weather","repository":"owner/weather","description":"Weather lookup","author":"dev","tags":["tooling"]},
			{"name":"summarize","repository":"owner/summarize","description":"Summaries","author":"dev2","tags":["text"]}
		]`))
	}))
	defer srv.Close()

	installer := &Installer{
		httpClient:     srv.Client(),
		skillsIndexURL: srv.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	items, err := installer.Search(ctx)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(items))
	}
	if items[0].Name != "weather" || items[0].Repository != "owner/weather" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

func TestInstallerSearch_Non200ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	installer := &Installer{
		httpClient:     srv.Client(),
		skillsIndexURL: srv.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := installer.Search(ctx); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
