package telegram

import (
	"context"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/voice"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestMarkdownToHTML_RendersBoldAndCode(t *testing.T) {
	out := markdownToHTML("**b** `c`")
	if strings.Contains(out, "&lt;b&gt;") {
		t.Fatalf("expected bold tags to be real HTML, got: %s", out)
	}
	if !strings.Contains(out, "<b>b</b>") {
		t.Fatalf("expected bold to render, got: %s", out)
	}
	if !strings.Contains(out, "<code>c</code>") {
		t.Fatalf("expected code to render, got: %s", out)
	}
}

func TestRenderMessageHTML_IncludesThinkContent(t *testing.T) {
	out := renderMessageHTML("<think>**t**</think>**m**")
	if strings.Contains(out, "<think>") {
		t.Fatalf("expected think tags removed, got: %s", out)
	}
	if !strings.Contains(out, "Thinking:") {
		t.Fatalf("expected thinking label, got: %s", out)
	}
	if !strings.Contains(out, "<b>t</b>") || !strings.Contains(out, "<b>m</b>") {
		t.Fatalf("expected rendered think and main, got: %s", out)
	}
}

func TestParseInt64_Valid(t *testing.T) {
	got, err := parseInt64("12345")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != 12345 {
		t.Fatalf("expected 12345, got %d", got)
	}
}

func TestParseInt64_Invalid(t *testing.T) {
	if _, err := parseInt64("not-a-number"); err == nil {
		t.Fatal("expected error for invalid chat id")
	}
}

type fakeTranscriber struct {
	text string
	err  error
	got  voice.Input
}

func (f *fakeTranscriber) Transcribe(ctx context.Context, input voice.Input) (string, error) {
	f.got = input
	return f.text, f.err
}

func TestHandleMessage_VoiceMessageUsesTranscriber(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ch := New(&config.TelegramConfig{}, msgBus, nil)

	ft := &fakeTranscriber{text: "voice text"}
	ch.transcriber = ft
	ch.downloadVoice = func(ctx context.Context, fileID, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessage(&tgbotapi.Message{
		MessageID: 7,
		From:      &tgbotapi.User{ID: 123, UserName: "alice"},
		Chat:      &tgbotapi.Chat{ID: 42},
		Voice:     &tgbotapi.Voice{FileID: "file-1", MimeType: "audio/ogg"},
	})

	select {
	case in := <-msgBus.Inbound():
		if in.Content != "voice text" {
			t.Fatalf("expected transcribed content, got %q", in.Content)
		}
		if in.Metadata["transcribed_audio"] != true {
			t.Fatalf("expected transcribed_audio metadata true, got %+v", in.Metadata)
		}
	default:
		t.Fatal("expected inbound message")
	}
}

func TestHandleMessage_TranscriptionFailureDoesNotDropText(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ch := New(&config.TelegramConfig{}, msgBus, nil)

	ft := &fakeTranscriber{err: context.DeadlineExceeded}
	ch.transcriber = ft
	ch.downloadVoice = func(ctx context.Context, fileID, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessage(&tgbotapi.Message{
		MessageID: 9,
		From:      &tgbotapi.User{ID: 555, UserName: "bob"},
		Chat:      &tgbotapi.Chat{ID: 77},
		Text:      "typed text",
		Voice:     &tgbotapi.Voice{FileID: "file-2", MimeType: "audio/ogg"},
	})

	select {
	case in := <-msgBus.Inbound():
		if in.Content != "typed text" {
			t.Fatalf("expected text content fallback, got %q", in.Content)
		}
	default:
		t.Fatal("expected inbound text message")
	}
}
