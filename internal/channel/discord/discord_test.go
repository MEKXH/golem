package discord

import (
	"context"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/config"
	"github.com/MEKXH/golem/internal/voice"
	"github.com/bwmarrin/discordgo"
)

type fakeTranscriber struct {
	text string
	err  error
	got  voice.Input
}

func (f *fakeTranscriber) Transcribe(ctx context.Context, input voice.Input) (string, error) {
	f.got = input
	return f.text, f.err
}

func TestHandleMessage_AudioAttachmentUsesTranscriber(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{text: "voice text"}
	ch := New(&config.DiscordConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessage(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "m1",
			GuildID:   "g1",
			ChannelID: "c1",
			Author:    &discordgo.User{ID: "u1", Username: "alice"},
			Attachments: []*discordgo.MessageAttachment{
				{URL: "https://cdn.discord.test/v.ogg", Filename: "v.ogg", ContentType: "audio/ogg"},
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
	default:
		t.Fatal("expected inbound message")
	}
}

func TestHandleMessage_TranscriptionFailureDoesNotDropText(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	ft := &fakeTranscriber{err: context.DeadlineExceeded}
	ch := New(&config.DiscordConfig{}, msgBus, ft)
	ch.downloadAudio = func(ctx context.Context, url, fileName, mimeType string) (voice.Input, error) {
		return voice.Input{
			FileName: fileName,
			MIMEType: mimeType,
			Data:     []byte("audio"),
		}, nil
	}

	ch.handleMessage(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "m2",
			GuildID:   "g1",
			ChannelID: "c2",
			Content:   "typed text",
			Author:    &discordgo.User{ID: "u2", Username: "bob"},
			Attachments: []*discordgo.MessageAttachment{
				{URL: "https://cdn.discord.test/v.ogg", Filename: "v.ogg", ContentType: "audio/ogg"},
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
