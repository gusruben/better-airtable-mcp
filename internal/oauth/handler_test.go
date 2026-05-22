package oauth

import (
	"context"
	"testing"
	"time"
)

func TestHandlerPruneExpiredStateRemovesExpiredEntries(t *testing.T) {
	now := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	handler := &Handler{
		now: nowFunc(now),
		authRequests: map[string]authorizationRequest{
			"expired-request": {ExpiresAt: now.Add(-time.Second)},
			"fresh-request":   {ExpiresAt: now.Add(time.Minute)},
		},
		authCodes: map[string]authorizationCode{
			"expired-code": {ExpiresAt: now.Add(-time.Second)},
			"fresh-code":   {ExpiresAt: now.Add(time.Minute)},
		},
	}

	removed := handler.PruneExpiredState()
	if removed != 2 {
		t.Fatalf("expected to remove 2 expired entries, removed %d", removed)
	}

	if _, ok := handler.authRequests["expired-request"]; ok {
		t.Fatal("expected expired auth request to be removed")
	}
	if _, ok := handler.authCodes["expired-code"]; ok {
		t.Fatal("expected expired auth code to be removed")
	}

	if _, ok := handler.authRequests["fresh-request"]; !ok {
		t.Fatal("expected fresh auth request to remain")
	}
	if _, ok := handler.authCodes["fresh-code"]; !ok {
		t.Fatal("expected fresh auth code to remain")
	}
}

func TestHandlerRunCleanupLoopPrunesExpiredState(t *testing.T) {
	now := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	handler := &Handler{
		now: nowFunc(now),
		authRequests: map[string]authorizationRequest{
			"expired-request": {ExpiresAt: now.Add(-time.Second)},
		},
		authCodes: map[string]authorizationCode{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		handler.RunCleanupLoop(ctx, 5*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(handler.authRequests) == 0 {
			cancel()
			<-done
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
	<-done
	t.Fatal("expected cleanup loop to prune expired auth request")
}

func nowFunc(now time.Time) func() time.Time {
	return func() time.Time {
		return now
	}
}
