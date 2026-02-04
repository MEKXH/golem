package session

import "testing"

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
