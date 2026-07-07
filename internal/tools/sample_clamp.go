package tools

import (
	"fmt"
	"strings"
)

const schemaSampleArrayMaxItems = 3

// clampSampleRows bounds the size of sample-row values before they are
// emitted in tool output. Raw sample rows can be arbitrarily large: a linked
// record cell may hold hundreds of record IDs, and attachment cells carry
// signed thumbnail URLs that are hundreds of characters each.
func clampSampleRows(rows []map[string]any) []map[string]any {
	clamped := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		clampedRow := make(map[string]any, len(row))
		for key, value := range row {
			clampedRow[key] = clampSampleValue(value)
		}
		clamped = append(clamped, clampedRow)
	}
	return clamped
}

func clampSampleValue(value any) any {
	switch typed := value.(type) {
	case string:
		return truncateSchemaSampleValue(typed)
	case []any:
		items := typed
		var marker any
		if len(items) > schemaSampleArrayMaxItems {
			marker = fmt.Sprintf("[+%d more, %d total]", len(items)-schemaSampleArrayMaxItems, len(items))
			items = items[:schemaSampleArrayMaxItems]
		}
		clamped := make([]any, 0, len(items)+1)
		for _, item := range items {
			clamped = append(clamped, clampSampleValue(item))
		}
		if marker != nil {
			clamped = append(clamped, marker)
		}
		return clamped
	case map[string]any:
		clamped := make(map[string]any, len(typed))
		dropBulkyAttachmentKeys := isAttachmentMap(typed)
		for key, item := range typed {
			if dropBulkyAttachmentKeys && (key == "url" || key == "thumbnails") {
				continue
			}
			clamped[key] = clampSampleValue(item)
		}
		return clamped
	default:
		return typed
	}
}

// isAttachmentMap detects Airtable attachment objects so their signed URLs
// (which expire within hours and dwarf the rest of the schema output) can be
// dropped from sample rows.
func isAttachmentMap(value map[string]any) bool {
	id, _ := value["id"].(string)
	if !strings.HasPrefix(id, "att") {
		return false
	}
	_, hasURL := value["url"]
	_, hasThumbnails := value["thumbnails"]
	_, hasFilename := value["filename"]
	return hasURL || hasThumbnails || hasFilename
}
