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

func TestStripUncreatableFieldsStripsCreateTableFields(t *testing.T) {
	input, warnings, guidance := stripUncreatableFields(SchemaInput{
		Base: "app123",
		Operations: []SchemaOperation{{
			Type: schemaOpCreateTable,
			Name: "Shop Orders",
			Fields: []SchemaFieldDef{
				{Name: "Email", Type: "email"},
				{Name: "created_at", Type: "createdTime"},
				{Name: "Prize", Type: "singleLineText"},
			},
		}},
	})
	if guidance != "" {
		t.Fatalf("unexpected guidance: %q", guidance)
	}
	fields := input.Operations[0].Fields
	if len(fields) != 2 || fields[0].Name != "Email" || fields[1].Name != "Prize" {
		t.Fatalf("expected createdTime field stripped, got %#v", fields)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "createdTime") || !strings.Contains(warnings[0], "created_at") {
		t.Fatalf("expected a warning naming the stripped field, got %#v", warnings)
	}
	if strings.Contains(warnings[0], "primary field") {
		t.Fatalf("primary-field note should only appear when the first field is stripped, got %q", warnings[0])
	}
}

func TestStripUncreatableFieldsNotesPrimaryFieldShift(t *testing.T) {
	_, warnings, guidance := stripUncreatableFields(SchemaInput{
		Base: "app123",
		Operations: []SchemaOperation{{
			Type: schemaOpCreateTable,
			Name: "Shop Orders",
			Fields: []SchemaFieldDef{
				{Name: "created_at", Type: "createdTime"},
				{Name: "Email", Type: "email"},
			},
		}},
	})
	if guidance != "" {
		t.Fatalf("unexpected guidance: %q", guidance)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "primary field") {
		t.Fatalf("expected primary-field note when first field is stripped, got %#v", warnings)
	}
}

func TestStripUncreatableFieldsDropsCreateFieldOp(t *testing.T) {
	input, warnings, guidance := stripUncreatableFields(SchemaInput{
		Base: "app123",
		Operations: []SchemaOperation{
			{Type: schemaOpCreateField, Table: "Shop Orders", Name: "Modified By", FieldType: "lastModifiedBy"},
			{Type: schemaOpCreateField, Table: "Shop Orders", Name: "Notes", FieldType: "multilineText"},
		},
	})
	if guidance != "" {
		t.Fatalf("unexpected guidance: %q", guidance)
	}
	if len(input.Operations) != 1 || input.Operations[0].Name != "Notes" {
		t.Fatalf("expected lastModifiedBy op dropped, got %#v", input.Operations)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "lastModifiedBy") {
		t.Fatalf("expected a warning for the dropped op, got %#v", warnings)
	}
}

func TestStripUncreatableFieldsGuidanceWhenNothingCreatable(t *testing.T) {
	cases := map[string]SchemaInput{
		"create_table with only uncreatable fields": {
			Base: "app123",
			Operations: []SchemaOperation{{
				Type:   schemaOpCreateTable,
				Name:   "Audit",
				Fields: []SchemaFieldDef{{Name: "Created", Type: "createdTime"}, {Name: "ID", Type: "autoNumber"}},
			}},
		},
		"all operations dropped": {
			Base: "app123",
			Operations: []SchemaOperation{
				{Type: schemaOpCreateField, Table: "Projects", Name: "Created By", FieldType: "createdBy"},
			},
		},
	}
	for name, in := range cases {
		_, _, guidance := stripUncreatableFields(in)
		if guidance == "" {
			t.Fatalf("expected guidance for %q", name)
		}
	}
}

func TestStripUncreatableFieldsPassesCleanInputThrough(t *testing.T) {
	in := SchemaInput{
		Base: "app123",
		Operations: []SchemaOperation{
			{Type: schemaOpCreateTable, Name: "Projects", Fields: []SchemaFieldDef{{Name: "Name", Type: "singleLineText"}}},
			{Type: schemaOpCreateField, Table: "Projects", Name: "Status", FieldType: "singleSelect"},
			{Type: schemaOpUpdateField, Table: "Projects", Field: "Status", Name: "State"},
		},
	}
	out, warnings, guidance := stripUncreatableFields(in)
	if guidance != "" || len(warnings) != 0 {
		t.Fatalf("clean input should pass through, got warnings=%#v guidance=%q", warnings, guidance)
	}
	if len(out.Operations) != 3 || len(out.Operations[0].Fields) != 1 {
		t.Fatalf("clean input was modified: %#v", out.Operations)
	}
}

func TestSchemaMutateCallRefusesAllUncreatableFields(t *testing.T) {
	tool := NewSchemaMutateTool(nil)
	result, err := tool.Call(context.Background(), json.RawMessage(`{
		"base": "app123",
		"operations": [{"type": "create_field", "table": "Projects", "name": "Created", "field_type": "createdTime"}]
	}`))
	if err != nil {
		t.Fatalf("Call() returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected refusal to be reported as an error result")
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].Text, "can't create") {
		t.Fatalf("expected uncreatable-type guidance in result, got %#v", result.Content)
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
