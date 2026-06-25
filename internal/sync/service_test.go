package syncer

import (
	"testing"
)

func TestBuildSyncPlansDisambiguatesDuplicateFieldNames(t *testing.T) {
	tables := []Table{{
		ID:   "tbl1",
		Name: "Tasks",
		Fields: []Field{
			{ID: "fld1", Name: "Status", Type: "singleLineText"},
			{ID: "fld2", Name: "Status", Type: "singleLineText"},
		},
	}}

	plans, err := buildSyncPlans(tables)
	if err != nil {
		t.Fatalf("buildSyncPlans() returned error: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	fields := plans[0].Table.Fields
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d (%#v)", len(fields), fields)
	}

	if fields[0].AirtableFieldID != "fld1" || fields[1].AirtableFieldID != "fld2" {
		t.Fatalf("field order not preserved: got %q, %q", fields[0].AirtableFieldID, fields[1].AirtableFieldID)
	}
	if fields[0].DuckDBColumnName == fields[1].DuckDBColumnName {
		t.Fatalf("duplicate field names collapsed to the same column %q", fields[0].DuckDBColumnName)
	}
	if got := fields[0].DuckDBColumnName; got != "status" {
		t.Fatalf("field fld1: want column %q, got %q", "status", got)
	}
	if got := fields[1].DuckDBColumnName; got != "status_2" {
		t.Fatalf("field fld2: want column %q, got %q", "status_2", got)
	}
}

func TestBuildSyncPlansKeepsSanitizedIndexAlignedAcrossOmittedFields(t *testing.T) {
	// "button" maps to an omitted DuckDB type, so it is dropped from the synced
	// fields. The sanitized-name index must skip it without drifting, otherwise
	// the fields straddling the omission get the wrong columns.
	tables := []Table{{
		ID:   "tbl1",
		Name: "Tasks",
		Fields: []Field{
			{ID: "fld1", Name: "Status", Type: "singleLineText"},
			{ID: "fldBtn", Name: "Open", Type: "button"},
			{ID: "fld2", Name: "Status", Type: "singleLineText"},
		},
	}}

	plans, err := buildSyncPlans(tables)
	if err != nil {
		t.Fatalf("buildSyncPlans() returned error: %v", err)
	}

	fields := plans[0].Table.Fields
	if len(fields) != 2 {
		t.Fatalf("expected the omitted button field to be dropped, got %d fields (%#v)", len(fields), fields)
	}

	want := map[string]string{
		"fld1": "status",
		"fld2": "status_2",
	}
	for _, field := range fields {
		if got := field.DuckDBColumnName; got != want[field.AirtableFieldID] {
			t.Fatalf("field %s: want column %q, got %q", field.AirtableFieldID, want[field.AirtableFieldID], got)
		}
	}
}
