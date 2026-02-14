package slack

import (
	"context"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/voice"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func TestParseChatID(t *testing.T) {
	channelID, threadTS := parseChatID("C123/1700000000.1")
	if channelID != "C123" || threadTS != "1700000000.1" {
		t.Fatalf("unexpected parse result: channel=%q thread=%q", channelID, threadTS)
	}
}

func TestParseChatID_ChannelOnly(t *testing.T) {
	channelID, threadTS := parseChatID("C123")
	if channelID != "C123" || threadTS != "" {
		t.Fatalf("unexpected parse result: channel=%q thread=%q", channelID, threadTS)
	}
}

type fakeTranscriber struct {
	text        string
	err         error
	got         voice.Input
	hasDeadline bool
	callCount   int
}

func (f *fakeTranscriber) Transcribe(ctx context.Context, input voice.Input) (string, error) {
	f.got = input
	_, f.hasDeadline = ctx.Deadline()
	f.callCount++
	return f.text, f.err
}

func TestHandleMessageEvent_AudioFileUsesTranscriber(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{text: "voice text"}
	ch := New(&config.SlackConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessageEvent(&slackevents.MessageEvent{
		User:      "U1",
		Text:      "hello",
		TimeStamp: "1700000000.1",
		Channel:   "C1",
		Message: &slack.Msg{
			Files: []slack.File{
				{Name: "voice.ogg", Mimetype: "audio/ogg", URLPrivateDownload: "https://files.slack.test/voice.ogg"},
			},
		},
	})

	select {
	case in := <-msgBus.Inbound():
		if !strings.Contains(in.Content, "[voice] voice text") {
			t.Fatalf("expected transcribed content, got %q", in.Content)
		}
		if in.Metadata["transcribed_audio"] != true {
			t.Fatalf("expected transcribed_audio metadata true, got %+v", in.Metadata)
		}
		if !ft.hasDeadline {
			t.Fatal("expected transcription context with deadline")
		}
	default:
		t.Fatal("expected inbound message")
	}
}

func TestHandleMessageEvent_TranscriptionFailureDoesNotDropText(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{err: context.DeadlineExceeded}
	ch := New(&config.SlackConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessageEvent(&slackevents.MessageEvent{
		User:      "U2",
		Text:      "typed text",
		TimeStamp: "1700000000.2",
		Channel:   "C2",
		Message: &slack.Msg{
			Files: []slack.File{
				{Name: "voice.ogg", Mimetype: "audio/ogg", URLPrivateDownload: "https://files.slack.test/voice.ogg"},
			},
		},
	})

	select {
	case in := <-msgBus.Inbound():
		if !strings.Contains(in.Content, "typed text") {
			t.Fatalf("expected text content retained, got %q", in.Content)
		}
	default:
		t.Fatal("expected inbound message")
	}
}

func TestHandleMessageEvent_MultipleAudioFilesTranscribed(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{text: "voice text"}
	ch := New(&config.SlackConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessageEvent(&slackevents.MessageEvent{
		User:      "U3",
		Text:      "",
		TimeStamp: "1700000000.3",
		Channel:   "C3",
		Message: &slack.Msg{
			Files: []slack.File{
				{Name: "voice1.ogg", Mimetype: "audio/ogg", URLPrivateDownload: "https://files.slack.test/voice1.ogg"},
				{Name: "voice2.ogg", Mimetype: "audio/ogg", URLPrivateDownload: "https://files.slack.test/voice2.ogg"},
			},
		},
	})

	select {
	case in := <-msgBus.Inbound():
		if ft.callCount != 2 {
			t.Fatalf("expected 2 transcription calls, got %d", ft.callCount)
		}
		if in.Metadata["transcribed_audio_count"] != 2 {
			t.Fatalf("expected transcribed_audio_count=2, got %+v", in.Metadata)
		}
	default:
		t.Fatal("expected inbound message")
	}
}

func TestHandleMessageEvent_AudioOnlyFailureKeepsAudioPlaceholder(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{err: context.DeadlineExceeded}
	ch := New(&config.SlackConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessageEvent(&slackevents.MessageEvent{
		User:      "U4",
		Text:      "",
		TimeStamp: "1700000000.4",
		Channel:   "C4",
		Message: &slack.Msg{
			Files: []slack.File{
				{Name: "voice4.ogg", Mimetype: "audio/ogg", URLPrivateDownload: "https://files.slack.test/voice4.ogg"},
			},
		},
	})

	select {
	case in := <-msgBus.Inbound():
		if !strings.Contains(in.Content, "[audio: voice4.ogg]") {
			t.Fatalf("expected audio placeholder, got %q", in.Content)
		}
	default:
		t.Fatal("expected inbound message")
	}
}
