package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSession_AddMessage(t *testing.T) {
	sess := &Session{Key: "test"}
	sess.AddMessage("user", "hello")
	sess.AddMessage("assistant", "hi there")

	history := sess.GetHistory(10)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("expected role=user, got %s", history[0].Role)
	}
}

func TestManager_GetOrCreate(t *testing.T) {
	mgr := NewManager(t.TempDir())

	sess1 := mgr.GetOrCreate("test:123")
	sess2 := mgr.GetOrCreate("test:123")

	if sess1 != sess2 {
		t.Error("expected same session instance")
	}
}

func TestSession_SaveAndLoad(t *testing.T) {
	baseDir := t.TempDir()

	// Create a manager, get a session, add messages, and save
	mgr1 := NewManager(baseDir)
	sess := mgr1.GetOrCreate("persist-test")
	sess.AddMessage("user", "What is Go?")
	sess.AddMessage("assistant", "Go is a programming language.")
	sess.AddMessage("user", "Tell me more.")

	if err := mgr1.Save(sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Create a NEW manager with the same base dir and load the same session
	mgr2 := NewManager(baseDir)
	loaded := mgr2.GetOrCreate("persist-test")

	history := loaded.GetHistory(0) // 0 means all
	if len(history) != 3 {
		t.Fatalf("expected 3 messages after load, got %d", len(history))
	}

	// Verify message content and roles
	if history[0].Role != "user" || history[0].Content != "What is Go?" {
		t.Errorf("message[0]: expected role=user content='What is Go?', got role=%s content=%s", history[0].Role, history[0].Content)
	}
	if history[1].Role != "assistant" || history[1].Content != "Go is a programming language." {
		t.Errorf("message[1]: expected role=assistant content='Go is a programming language.', got role=%s content=%s", history[1].Role, history[1].Content)
	}
	if history[2].Role != "user" || history[2].Content != "Tell me more." {
		t.Errorf("message[2]: expected role=user content='Tell me more.', got role=%s content=%s", history[2].Role, history[2].Content)
	}
}

func TestSession_EmptySessionNotSaved(t *testing.T) {
	baseDir := t.TempDir()

	mgr := NewManager(baseDir)
	sess := mgr.GetOrCreate("empty-session")

	// Save with no messages - should not create a file
	if err := mgr.Save(sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Check that no session file was created
	sessDir := filepath.Join(baseDir, "sessions")
	entries, err := os.ReadDir(sessDir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "empty-session.jsonl" {
			t.Fatal("expected no file for empty session, but file was created")
		}
	}
}

func TestSession_SaveAndLoad_LongMessage(t *testing.T) {
	baseDir := t.TempDir()
	mgr1 := NewManager(baseDir)
	sess := mgr1.GetOrCreate("long-message")
	longText := strings.Repeat("x", 200000)
	sess.AddMessage("user", longText)

	if err := mgr1.Save(sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	mgr2 := NewManager(baseDir)
	loaded := mgr2.GetOrCreate("long-message")
	history := loaded.GetHistory(0)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after load, got %d", len(history))
	}
	if history[0].Content != longText {
		t.Fatalf("expected long message content round-trip, got length %d", len(history[0].Content))
	}
}

func TestSession_AppendAndLoad(t *testing.T) {
	baseDir := t.TempDir()
	mgr1 := NewManager(baseDir)
	sess := mgr1.GetOrCreate("append-test")

	// Initial save with one message
	msg1 := sess.AddMessage("user", "Hello")
	mgr1.Save(sess)
	// Just to use msg1
	if msg1.Content != "Hello" {
		t.Errorf("msg1 content mismatch")
	}

	// Append two more messages
	msg2 := sess.AddMessage("assistant", "Hi")
	msg3 := sess.AddMessage("user", "How are you?")
	if err := mgr1.Append(sess.Key, msg2, msg3); err != nil {
		t.Fatalf("Append error: %v", err)
	}

	// Load in a new manager
	mgr2 := NewManager(baseDir)
	loaded := mgr2.GetOrCreate("append-test")
	history := loaded.GetHistory(0)

	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	if history[0].Content != "Hello" {
		t.Errorf("msg 0 content mismatch")
	}
	if history[1].Content != "Hi" {
		t.Errorf("msg 1 content mismatch")
	}
	if history[2].Content != "How are you?" {
		t.Errorf("msg 2 content mismatch")
	}
}
