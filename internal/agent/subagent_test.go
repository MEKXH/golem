package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MEKXH/golem/internal/bus"
	"github.com/MEKXH/golem/internal/tools"
)

type fakeSubagentProcessor struct {
	response string
	err      error
	lastCall struct {
		channel   string
		chatID    string
		senderID  string
		sessionID string
		content   string
	}
}

func (f *fakeSubagentProcessor) ProcessForChannelWithSession(ctx context.Context, channel, chatID, senderID, sessionID, content string) (string, error) {
	f.lastCall.channel = channel
	f.lastCall.chatID = chatID
	f.lastCall.senderID = senderID
	f.lastCall.sessionID = sessionID
	f.lastCall.content = content
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func TestSubagentManager_SpawnPublishesSystemResult(t *testing.T) {
	msgBus := bus.NewMessageBus(2)
	processor := &fakeSubagentProcessor{response: "task done"}
	manager := NewSubagentManager(msgBus, processor, 2*time.Second)

	taskID, err := manager.Spawn(context.Background(), tools.SubagentRequest{
		Task:           "collect diagnostics",
		Label:          "diag",
		OriginChannel:  "telegram",
		OriginChatID:   "1001",
		OriginSenderID: "bob",
		RequestID:      "req-9",
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if taskID == "" {
		t.Fatal("expected non-empty task id")
	}

	select {
	case msg := <-msgBus.Inbound():
		if msg.Channel != bus.SystemChannel {
			t.Fatalf("expected system channel, got %q", msg.Channel)
		}
		if msg.Metadata[bus.SystemMetaType] != bus.SystemTypeSubagentResult {
			t.Fatalf("unexpected system type metadata: %+v", msg.Metadata)
		}
		if msg.Metadata[bus.SystemMetaOriginChannel] != "telegram" || msg.Metadata[bus.SystemMetaOriginChatID] != "1001" {
			t.Fatalf("unexpected origin metadata: %+v", msg.Metadata)
		}
		if msg.Metadata[bus.SystemMetaTaskID] == "" {
			t.Fatalf("expected task id metadata, got %+v", msg.Metadata)
		}
		if msg.RequestID != "req-9" {
			t.Fatalf("expected request id propagation, got %q", msg.RequestID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subagent system message")
	}
}

func TestSubagentManager_RunSyncReturnsError(t *testing.T) {
	msgBus := bus.NewMessageBus(1)
	processor := &fakeSubagentProcessor{err: errors.New("boom")}
	manager := NewSubagentManager(msgBus, processor, 2*time.Second)

	_, err := manager.RunSync(context.Background(), tools.SubagentRequest{
		Task:           "failing task",
		OriginChannel:  "discord",
		OriginChatID:   "x",
		OriginSenderID: "alice",
	})
	if err == nil {
		t.Fatal("expected sync run error")
	}
}
