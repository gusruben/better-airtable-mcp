package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestListBasesPaginatesBeyondFirstPage reproduces the base-count-crossed-1000
// bug: Airtable's /v0/meta/bases returns at most 1000 bases per page plus an
// "offset" for the next page. ListBases must follow that offset, or every base
// past #1000 silently disappears — and since resolveBase requires a base to be
// present in the ListBases result, those bases become unreachable even by exact
// ID ("base ... was not found"), on an "all bases" grant.
//
// Here the grant has 1002 bases; the target sits at position 1001. Against the
// unpaginated code this FAILS (returns 1000, target absent).
func TestListBasesPaginatesBeyondFirstPage(t *testing.T) {
	const targetID = "appTarget1001"

	page1 := make([]map[string]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		page1 = append(page1, map[string]any{
			"id":              fmt.Sprintf("appPage1_%04d", i),
			"name":            fmt.Sprintf("Base %04d", i),
			"permissionLevel": "create",
		})
	}
	page2 := []map[string]any{
		{"id": targetID, "name": "Restricted Base", "permissionLevel": "create"},
		{"id": "appPage2_extra", "name": "Base 1002", "permissionLevel": "create"},
	}

	var basesCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/meta/bases" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		basesCalls++
		switch r.URL.Query().Get("offset") {
		case "":
			writeManagerJSON(t, w, map[string]any{"bases": page1, "offset": "page2"})
		case "page2":
			writeManagerJSON(t, w, map[string]any{"bases": page2})
		default:
			t.Fatalf("unexpected offset %q", r.URL.Query().Get("offset"))
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, server.Client())
	bases, err := client.ListBases(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListBases() error: %v", err)
	}

	t.Logf("bases returned: %d (server pages served: %d)", len(bases), basesCalls)

	if len(bases) != 1002 {
		t.Fatalf("truncated base list: got %d bases, want 1002 (offset pagination not followed)", len(bases))
	}

	found := false
	for _, b := range bases {
		if b.ID == targetID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("base at position 1001 (%s) missing from list — unreachable by resolveBase", targetID)
	}
}
