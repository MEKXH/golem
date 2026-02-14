package voice

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAITranscriber_TranscribeSuccess(t *testing.T) {
	var gotAuth string
	var gotModel string
	var gotFile string
	var gotMime string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("expected multipart content type, got %q", r.Header.Get("Content-Type"))
		}

		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		gotModel = r.FormValue("model")

		f, fh, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile(file): %v", err)
		}
		defer f.Close()
		gotFile = fh.Filename
		gotMime = fh.Header.Get("Content-Type")
		raw, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if string(raw) != "audio-bytes" {
			t.Fatalf("unexpected audio payload: %q", string(raw))
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"text": "hello from audio"})
	}))
	defer srv.Close()

	tr, err := NewOpenAITranscriber("key-1", srv.URL+"/v1", "gpt-4o-mini-transcribe", 5*time.Second)
	if err != nil {
		t.Fatalf("NewOpenAITranscriber error: %v", err)
	}

	text, err := tr.Transcribe(context.Background(), Input{
		FileName: "voice.ogg",
		MIMEType: "audio/ogg",
		Data:     []byte("audio-bytes"),
	})
	if err != nil {
		t.Fatalf("Transcribe error: %v", err)
	}
	if text != "hello from audio" {
		t.Fatalf("expected text from server, got %q", text)
	}
	if gotAuth != "Bearer key-1" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
	if gotModel != "gpt-4o-mini-transcribe" {
		t.Fatalf("expected model field, got %q", gotModel)
	}
	if gotFile != "voice.ogg" {
		t.Fatalf("expected filename voice.ogg, got %q", gotFile)
	}
	if gotMime != "audio/ogg" {
		t.Fatalf("expected mime audio/ogg, got %q", gotMime)
	}
}

func TestOpenAITranscriber_TranscribeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()

	tr, err := NewOpenAITranscriber("bad-key", srv.URL+"/v1", "gpt-4o-mini-transcribe", 5*time.Second)
	if err != nil {
		t.Fatalf("NewOpenAITranscriber error: %v", err)
	}

	_, err = tr.Transcribe(context.Background(), Input{
		FileName: "voice.ogg",
		MIMEType: "audio/ogg",
		Data:     []byte("audio-bytes"),
	})
	if err == nil {
		t.Fatal("expected error on non-2xx response")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status code in error, got: %v", err)
	}
}

func TestOpenAITranscriber_RejectsEmptyAudio(t *testing.T) {
	tr, err := NewOpenAITranscriber("k", "https://api.openai.com/v1", "gpt-4o-mini-transcribe", 5*time.Second)
	if err != nil {
		t.Fatalf("NewOpenAITranscriber error: %v", err)
	}

	_, err = tr.Transcribe(context.Background(), Input{
		FileName: "voice.ogg",
		MIMEType: "audio/ogg",
		Data:     nil,
	})
	if err == nil {
		t.Fatal("expected empty audio error")
	}
}

func TestOpenAITranscriber_RejectsTooLargeAudio(t *testing.T) {
	tr, err := NewOpenAITranscriber("k", "https://api.openai.com/v1", "gpt-4o-mini-transcribe", 5*time.Second)
	if err != nil {
		t.Fatalf("NewOpenAITranscriber error: %v", err)
	}

	tooLarge := make([]byte, maxInputBytes+1)
	_, err = tr.Transcribe(context.Background(), Input{
		FileName: "voice.ogg",
		MIMEType: "audio/ogg",
		Data:     tooLarge,
	})
	if err == nil {
		t.Fatal("expected too-large audio error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "too large") {
		t.Fatalf("expected too-large error, got: %v", err)
	}
}

func TestCreateMultipartForm_IncludesAudioFile(t *testing.T) {
	body, contentType, err := createMultipartForm(Input{
		FileName: "audio.mp3",
		MIMEType: "audio/mpeg",
		Data:     []byte("abc"),
	}, "model-x")
	if err != nil {
		t.Fatalf("createMultipartForm error: %v", err)
	}
	if !strings.Contains(contentType, "multipart/form-data") {
		t.Fatalf("expected multipart content type, got %q", contentType)
	}

	reader := multipart.NewReader(strings.NewReader(body.String()), strings.TrimPrefix(contentType, "multipart/form-data; boundary="))
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("NextPart(file): %v", err)
	}
	if part.FormName() != "file" {
		t.Fatalf("expected first part file, got %q", part.FormName())
	}
}
