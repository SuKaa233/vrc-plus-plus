package gamelog

import "testing"

func TestParsePlayerAndLocation(t *testing.T) {
	event, ok := ParseLine("output_log.txt", 12, `2026.08.15 11:10:22 Log - [Behaviour] OnPlayerJoined Alice (usr_123)`)
	if !ok || event.Type != "game.player-joined" {
		t.Fatalf("unexpected event: %#v", event)
	}
	event, ok = ParseLine("output_log.txt", 42, `2026.08.15 11:10:22 Log - [Behaviour] Joining wrld_abc:123~region(jp)~nonce(SECRET)`)
	if !ok || event.Type != "game.location" || string(event.Content) == "" {
		t.Fatalf("unexpected location: %#v", event)
	}
}
