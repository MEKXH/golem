package slack

import "testing"

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
