package vrchat

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestLimiterHonorsBlock(t *testing.T) {
	t.Parallel()
	limiter := newRequestLimiter(100, 1)
	limiter.Block(35 * time.Millisecond)
	started := time.Now()
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 25*time.Millisecond {
		t.Fatalf("Wait() returned too early after %s", elapsed)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	if got := parseRetryAfter("12", time.Now()); got != 12*time.Second {
		t.Fatalf("parseRetryAfter(seconds) = %s", got)
	}
	now := time.Now().UTC().Truncate(time.Second)
	header := now.Add(30 * time.Second).Format(http.TimeFormat)
	if got := parseRetryAfter(header, now); got != 30*time.Second {
		t.Fatalf("parseRetryAfter(date) = %s", got)
	}
}
