package syncer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateFieldStripsReadOnlyLinkOptions ensures that read-only link-field
// options (returned on read but rejected by Airtable's create API with
// INVALID_FIELD_TYPE_OPTIONS_FOR_CREATE) are dropped before the create call,
// while create-legal options like linkedTableId are preserved.
func TestCreateFieldStripsReadOnlyLinkOptions(t *testing.T) {
	var gotOptions map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Options map[string]any `json:"options"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		gotOptions = payload.Options
		writeManagerJSON(t, w, map[string]any{"id": "fldNew", "name": "user", "type": "multipleRecordLinks"})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, server.Client())
	_, err := client.CreateField(context.Background(), "token", "appX", "tblX", FieldDefinition{
		Name: "user",
		Type: "multipleRecordLinks",
		Options: map[string]any{
			"linkedTableId":           "tblLinked",
			"isReversed":              false,
			"prefersSingleRecordLink": true,
			"inverseLinkFieldId":      "fldInverse",
		},
	})
	if err != nil {
		t.Fatalf("CreateField() error: %v", err)
	}

	if _, ok := gotOptions["linkedTableId"]; !ok {
		t.Errorf("linkedTableId was dropped; options sent: %#v", gotOptions)
	}
	for _, k := range []string{"isReversed", "prefersSingleRecordLink", "inverseLinkFieldId"} {
		if _, present := gotOptions[k]; present {
			t.Errorf("read-only option %q should have been stripped; options sent: %#v", k, gotOptions)
		}
	}
}

// TestSanitizeCreateOptionsLeavesOtherTypesAlone confirms non-link field options
// pass through untouched.
func TestSanitizeCreateOptionsLeavesOtherTypesAlone(t *testing.T) {
	opts := map[string]any{"icon": "check", "color": "greenBright"}
	got := sanitizeCreateOptions("checkbox", opts)
	if len(got) != 2 || got["icon"] != "check" || got["color"] != "greenBright" {
		t.Fatalf("checkbox options were altered: %#v", got)
	}
}
