package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestManagerHotLoopsOnPersistentStorageFailure reproduces the backoff-free
// retry loop: when a sync fails and no snapshot file exists yet, the worker
// re-runs immediately with no delay, hammering the Airtable schema endpoint.
//
// Repro of the prod incident where an unwritable /data/duckdb (root-owned PVC
// mount) made every run fail with "create duckdb data dir: ... permission
// denied", and the worker spun ~5x/sec against Airtable's API (429 storm).
//
// Expected once a failure backoff is added: only a small, bounded number of
// schema fetches in the window. Against current code this FAILS, printing the
// real storm count.
func TestManagerHotLoopsOnPersistentStorageFailure(t *testing.T) {
	var schemaCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v0/meta/bases":
			writeManagerJSON(t, w, map[string]any{
				"bases": []map[string]any{{"id": "appProjects", "name": "Project Tracker", "permissionLevel": "create"}},
			})
		case "/v0/meta/bases/appProjects/tables":
			schemaCalls.Add(1)
			writeManagerJSON(t, w, map[string]any{
				"tables": []map[string]any{
					{
						"id":   "tblProjects",
						"name": "Projects",
						"fields": []map[string]any{
							{"id": "fldName", "name": "Name", "type": "singleLineText"},
						},
					},
				},
			})
		default:
			// records endpoint should never be reached: storage init fails first.
			t.Errorf("unexpected path %q (storage should fail before records)", r.URL.Path)
		}
	}))
	defer server.Close()

	// Make the DuckDB data dir impossible to create: put it under a parent with
	// no write bit, so os.MkdirAll(dataDir) fails with permission denied.
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	dataDir := filepath.Join(parent, "duckdb")

	service := NewService(NewHTTPClient(server.URL, server.Client()), dataDir)
	manager := NewManager(service, nil, staticTokenSource{}, time.Minute, time.Minute)

	if _, err := manager.RequestSync(context.Background(), "user_1", "Project Tracker"); err != nil {
		t.Fatalf("RequestSync() returned error: %v", err)
	}

	const window = 300 * time.Millisecond
	time.Sleep(window)
	got := schemaCalls.Load()

	t.Logf("schema fetches in %s: %d (%.0f/sec)", window, got, float64(got)/window.Seconds())

	const maxReasonable = 3
	if got > maxReasonable {
		t.Fatalf("backoff-free hot loop: %d schema fetches in %s (want <= %d with a failure backoff)", got, window, maxReasonable)
	}
}
