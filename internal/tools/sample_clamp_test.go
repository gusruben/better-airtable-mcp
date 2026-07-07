package tools

import (
	"strings"
	"testing"
)

func TestClampSampleRowsTruncatesLongStrings(t *testing.T) {
	rows := clampSampleRows([]map[string]any{
		{"token": strings.Repeat("a", 1000)},
	})

	got, ok := rows[0]["token"].(string)
	if !ok {
		t.Fatalf("expected string value, got %T", rows[0]["token"])
	}
	if len([]rune(got)) != schemaSampleValueMaxChars {
		t.Fatalf("expected clamped length %d, got %d", schemaSampleValueMaxChars, len([]rune(got)))
	}
	if !strings.HasSuffix(got, schemaSampleTruncationTag) {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func TestClampSampleRowsCapsLinkedRecordArrays(t *testing.T) {
	links := make([]any, 452)
	for index := range links {
		links[index] = "recAAAAAAAAAAAAAA"
	}

	rows := clampSampleRows([]map[string]any{
		{"sign_up_records": links},
	})

	got, ok := rows[0]["sign_up_records"].([]any)
	if !ok {
		t.Fatalf("expected array value, got %T", rows[0]["sign_up_records"])
	}
	if len(got) != schemaSampleArrayMaxItems+1 {
		t.Fatalf("expected %d items plus marker, got %d", schemaSampleArrayMaxItems, len(got))
	}
	if got[schemaSampleArrayMaxItems] != "[+449 more, 452 total]" {
		t.Fatalf("expected overflow marker, got %v", got[schemaSampleArrayMaxItems])
	}
}

func TestClampSampleRowsStripsAttachmentURLs(t *testing.T) {
	rows := clampSampleRows([]map[string]any{
		{
			"screenshot": []any{
				map[string]any{
					"id":       "attOmGOC29f5jB5qP",
					"filename": "ceiling.gif",
					"size":     float64(6642),
					"width":    float64(356),
					"height":   float64(349),
					"url":      "https://v5.airtableusercontent.com/" + strings.Repeat("x", 300),
					"thumbnails": map[string]any{
						"full": map[string]any{"url": strings.Repeat("y", 300)},
					},
				},
			},
		},
	})

	attachments, ok := rows[0]["screenshot"].([]any)
	if !ok || len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %v", rows[0]["screenshot"])
	}
	attachment, ok := attachments[0].(map[string]any)
	if !ok {
		t.Fatalf("expected attachment map, got %T", attachments[0])
	}
	if _, hasURL := attachment["url"]; hasURL {
		t.Fatalf("expected url to be dropped, got %v", attachment)
	}
	if _, hasThumbnails := attachment["thumbnails"]; hasThumbnails {
		t.Fatalf("expected thumbnails to be dropped, got %v", attachment)
	}
	if attachment["filename"] != "ceiling.gif" || attachment["id"] != "attOmGOC29f5jB5qP" {
		t.Fatalf("expected filename and id preserved, got %v", attachment)
	}
}

func TestClampSampleRowsKeepsURLsOnNonAttachmentMaps(t *testing.T) {
	rows := clampSampleRows([]map[string]any{
		{
			"metadata": map[string]any{
				"id":  "recSomething",
				"url": "https://example.com",
			},
		},
	})

	metadata, ok := rows[0]["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected map value, got %T", rows[0]["metadata"])
	}
	if metadata["url"] != "https://example.com" {
		t.Fatalf("expected url preserved on non-attachment map, got %v", metadata)
	}
}
