package gamelog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParsePlayerAndLocation(t *testing.T) {
	event, ok := ParseLine("output_log.txt", 12, `2026.08.15 11:10:22 Log - [Behaviour] OnPlayerJoined Alice (usr_123)`)
	if !ok || event.Type != "game.player-joined" {
		t.Fatalf("unexpected event: %#v", event)
	}
	event, ok = ParseLine("output_log.txt", 42, `2026.08.15 11:10:22 Log - [Behaviour] Joining wrld_abc:123~region(jp)~nonce(SECRET)`)
	if !ok || event.Type != "game.location" || string(event.Content) == "" {
		t.Fatalf("unexpected location: %#v", event)
	}
	if event.SensitiveLocation != "wrld_abc:123~region(jp)~nonce(SECRET)" {
		t.Fatalf("SensitiveLocation = %q", event.SensitiveLocation)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "SECRET") {
		t.Fatal("serialized event leaked the instance nonce")
	}
	event, ok = ParseLine("output_log.txt", 99, `2026.08.15 12:00:00 Debug - VRCApplication: HandleApplicationQuit at 123.4`)
	if !ok || event.Type != "game.quit-clean" {
		t.Fatalf("unexpected quit event: %#v", event)
	}
}
