package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSchemaInputAcceptsValidOperations(t *testing.T) {
	cases := []SchemaOperation{
		{Type: schemaOpCreateTable, Name: "Projects", Fields: []SchemaFieldDef{{Name: "Name", Type: "singleLineText"}}},
		{Type: schemaOpCreateField, Table: "Projects", Name: "Status", FieldType: "singleSelect"},
		{Type: schemaOpUpdateTable, Table: "Projects", Name: "Active Projects"},
		{Type: schemaOpUpdateField, Table: "Projects", Field: "Status", Name: "State"},
	}
	for _, op := range cases {
		if err := validateSchemaInput(SchemaInput{Base: "app123", Operations: []SchemaOperation{op}}); err != nil {
			t.Fatalf("validateSchemaInput(%s) returned error: %v", op.Type, err)
		}
	}
}

func TestValidateSchemaInputRejectsMissingFields(t *testing.T) {
	cases := map[string]SchemaOperation{
		"create_table without name":   {Type: schemaOpCreateTable, Fields: []SchemaFieldDef{{Name: "Name", Type: "singleLineText"}}},
		"create_table without fields": {Type: schemaOpCreateTable, Name: "Projects"},
		"create_field without type":   {Type: schemaOpCreateField, Table: "Projects", Name: "Status"},
		"update_table without change": {Type: schemaOpUpdateTable, Table: "Projects"},
		"update_field without target": {Type: schemaOpUpdateField, Table: "Projects", Name: "State"},
		"unknown type":                {Type: "frobnicate_table", Table: "Projects"},
	}
	for name, op := range cases {
		if err := validateSchemaInput(SchemaInput{Base: "app123", Operations: []SchemaOperation{op}}); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestUnsupportedSchemaIntentRefusesDeletion(t *testing.T) {
	for _, opType := range []string{"delete_field", "delete_table", "remove_field", "drop_table"} {
		guidance, ok := unsupportedSchemaIntent([]SchemaOperation{{Type: opType, Table: "Projects"}})
		if !ok {
			t.Fatalf("expected %q to be refused as unsupported", opType)
		}
		if !strings.Contains(guidance, "can't delete") {
			t.Fatalf("expected deletion guidance for %q, got %q", opType, guidance)
		}
	}
}

func TestUnsupportedSchemaIntentRefusesRetype(t *testing.T) {
	cases := []SchemaOperation{
		{Type: "change_field_type", Table: "Projects", Field: "Status"},
		{Type: "convert_field", Table: "Projects", Field: "Status"},
		{Type: schemaOpUpdateField, Table: "Projects", Field: "Status", FieldType: "number"},
	}
	for _, op := range cases {
		guidance, ok := unsupportedSchemaIntent([]SchemaOperation{op})
		if !ok {
			t.Fatalf("expected %q (field_type=%q) to be refused", op.Type, op.FieldType)
		}
		if !strings.Contains(guidance, "can't change") {
			t.Fatalf("expected retype guidance, got %q", guidance)
		}
	}
}

func TestUnsupportedSchemaIntentAllowsSupportedOps(t *testing.T) {
	if _, ok := unsupportedSchemaIntent([]SchemaOperation{
		{Type: schemaOpUpdateField, Table: "Projects", Field: "Status", Name: "State"},
		{Type: schemaOpCreateField, Table: "Projects", Name: "Owner", FieldType: "singleLineText"},
	}); ok {
		t.Fatal("supported operations should not be flagged as unsupported")
	}
}

func TestSchemaMutateCallRefusesDeletionWithGuidance(t *testing.T) {
	tool := NewSchemaMutateTool(nil)
	result, err := tool.Call(context.Background(), json.RawMessage(`{
		"base": "app123",
		"operations": [{"type": "delete_field", "table": "Projects", "field": "Status"}]
	}`))
	if err != nil {
		t.Fatalf("Call() returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected refusal to be reported as an error result")
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].Text, "can't delete") {
		t.Fatalf("expected deletion guidance in result, got %#v", result.Content)
	}
}
