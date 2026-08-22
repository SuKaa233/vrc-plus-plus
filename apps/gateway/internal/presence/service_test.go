package presence

import (
	"testing"
	"time"

	"github.com/SuKaa233/vrc-plus-plus/apps/gateway/internal/model"
)

func TestQuietHoursAcrossMidnight(t *testing.T) {
	rule := model.PresenceWatchRule{QuietStart: "23:00", QuietEnd: "08:00"}
	for _, item := range []struct {
		hour int
		want bool
	}{{22, false}, {23, true}, {3, true}, {8, false}} {
		got := inQuietHours(rule, time.Date(2026, 8, 20, item.hour, 0, 0, 0, time.Local))
		if got != item.want {
			t.Fatalf("hour %d = %v, want %v", item.hour, got, item.want)
		}
	}
}

func TestLocationClassification(t *testing.T) {
	if !joinableLocation("wrld_test:123~friends(usr_owner)") {
		t.Fatal("friends instance should be treated as joinable for a friend")
	}
	if joinableLocation("wrld_test:123~private(usr_owner)") {
		t.Fatal("private instance must not be treated as joinable")
	}
	if got := sanitizeLocation("wrld_test:123~private(usr_owner)~nonce(secret)"); got != "wrld_test:private" {
		t.Fatalf("sanitizeLocation() = %q", got)
	}
}
