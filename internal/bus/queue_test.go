package bus

import (
	"testing"
	"time"
)

func TestMessageBus_PublishConsume(t *testing.T) {
	bus := NewMessageBus(10)

	msg := &InboundMessage{
		Channel: "test",
		Content: "hello",
	}

	bus.PublishInbound(msg)

	select {
	case received := <-bus.Inbound():
		if received.Content != "hello" {
			t.Errorf("got Content=%q, want %q", received.Content, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}
