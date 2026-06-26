package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hackclub/better-airtable-mcp/internal/approval"
	"github.com/hackclub/better-airtable-mcp/internal/logx"
	"github.com/hackclub/better-airtable-mcp/internal/mcp"
)

// SchemaInput is the argument shape for the manage_schema tool. It only exposes
// operations Airtable's public meta API actually supports: creating tables and
// fields, and renaming/redescribing tables and fields. Deleting tables or
// fields and changing a field's type are not possible via the API, so this tool
// refuses those requests with guidance instead of faking them (for example, it
// never quietly renames a field to "deprecated" and calls it a deletion).
type SchemaInput struct {
	Base       string            `json:"base"`
	Operations []SchemaOperation `json:"operations"`
}

type SchemaOperation struct {
	Type        string           `json:"type"`
	Table       string           `json:"table,omitempty"`
	Field       string           `json:"field,omitempty"`
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	FieldType   string           `json:"field_type,omitempty"`
	Options     map[string]any   `json:"options,omitempty"`
	Fields      []SchemaFieldDef `json:"fields,omitempty"`
}

type SchemaFieldDef struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

const (
	schemaOpCreateTable = "create_table"
	schemaOpCreateField = "create_field"
	schemaOpUpdateTable = "update_table"
	schemaOpUpdateField = "update_field"

	// schemaUnsupportedGuidance is returned verbatim to the agent whenever it
	// asks for something Airtable's API can't do. It states the limitation
	// plainly and points at the honest alternatives the agent can choose.
	schemaDeleteGuidance = "Airtable's API can't delete tables or fields. This tool won't fake a deletion by renaming or clearing things behind your back. If you want to retire a field or table, do it by hand in Airtable, or (if you choose to) rename it with update_table/update_field to mark it deprecated."
	schemaRetypeGuidance = "Airtable's API can't change an existing field's type (update_field only changes name and description). To effectively change a type, create a new field with the type you want via create_field, migrate the values into it with the mutate tool, then remove the old field by hand in Airtable."
)

// SchemaMutateTool implements the manage_schema MCP tool.
type SchemaMutateTool struct {
	runtime *Runtime
}

func NewSchemaMutateTool(runtime *Runtime) mcp.Tool {
	return SchemaMutateTool{runtime: runtime}
}

func (SchemaMutateTool) Definition() mcp.ToolDefinition {
	return mcp.ToolDefinition{
		Name: "manage_schema",
		Description: "Request a schema change (create or rename tables and fields), subject to human approval. " +
			"Airtable's API cannot delete tables/fields or change a field's type; this tool will tell you so rather than fake it.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"base": map[string]any{
					"type":        "string",
					"description": "Airtable base ID or base name.",
				},
				"operations": map[string]any{
					"type":        "array",
					"description": "Schema operations to submit as a single approval request.",
					"minItems":    1,
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type": "string",
								"enum": []string{
									schemaOpCreateTable,
									schemaOpCreateField,
									schemaOpUpdateTable,
									schemaOpUpdateField,
								},
							},
							"table": map[string]any{
								"type":        "string",
								"description": "Existing table name or ID. Required for create_field, update_table, and update_field. Omit for create_table.",
							},
							"field": map[string]any{
								"type":        "string",
								"description": "Existing field name or ID to rename. Required for update_field.",
							},
							"name": map[string]any{
								"type":        "string",
								"description": "For create_table: the new table's name. For create_field: the new field's name. For update_table/update_field: the new name to set.",
							},
							"description": map[string]any{
								"type":        "string",
								"description": "Optional description to set on the table or field.",
							},
							"field_type": map[string]any{
								"type":        "string",
								"description": "Airtable field type for create_field (for example singleLineText, number, singleSelect, multipleRecordLinks).",
							},
							"options": map[string]any{
								"type":        "object",
								"description": "Airtable field-options object for create_field (choices for selects, linkedTableId for links, etc.).",
							},
							"fields": map[string]any{
								"type":        "array",
								"description": "For create_table: the fields to create. The first field becomes the table's primary field.",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"name":        map[string]any{"type": "string"},
										"type":        map[string]any{"type": "string"},
										"description": map[string]any{"type": "string"},
										"options":     map[string]any{"type": "object"},
									},
									"required": []string{"name", "type"},
								},
							},
						},
						"required": []string{"type"},
					},
				},
			},
			"required":             []string{"base", "operations"},
			"additionalProperties": false,
		},
	}
}

func (t SchemaMutateTool) Call(ctx context.Context, raw json.RawMessage) (mcp.ToolCallResult, error) {
	var input SchemaInput
	if err := decodeArgs(raw, &input); err != nil {
		return mcp.ToolCallResult{}, err
	}

	// Honest-refusal pass: if the agent is asking for something Airtable's API
	// can't do, explain it and stop, rather than validating it as a normal op.
	if guidance, ok := unsupportedSchemaIntent(input.Operations); ok {
		return mcp.ErrorResult(guidance, map[string]any{
			"reason": "unsupported_schema_operation",
		}), nil
	}

	if err := validateSchemaInput(input); err != nil {
		return mcp.ToolCallResult{}, err
	}

	if t.runtime == nil || t.runtime.Approval == nil {
		return mcp.ErrorResult("manage_schema approval flow is not configured; payload validation passed", map[string]any{
			"base":            strings.TrimSpace(input.Base),
			"operation_count": len(input.Operations),
			"operation_types": collectSchemaOperationTypes(input.Operations),
		}), nil
	}

	userID, ok := authenticatedUserID(ctx)
	if !ok {
		err := fmt.Errorf("missing authenticated user")
		logToolFailed(ctx, "manage_schema", err)
		return mcp.ToolCallResult{}, err
	}

	prepared, err := t.runtime.Approval.PrepareSchemaMutation(ctx, userID, toSchemaApprovalRequest(ctx, input))
	if err != nil {
		logToolFailed(ctx, "manage_schema", err, "user_id", userID)
		return mcp.ToolCallResult{}, err
	}

	payload := map[string]any{
		"operation_id":          prepared.OperationID,
		"status":                prepared.Status,
		"approval_url":          prepared.ApprovalURL,
		"expires_at":            prepared.ExpiresAt.Format(time.RFC3339),
		"summary":               prepared.Summary,
		"assistant_instruction": approvalURLAssistantInstruction,
	}
	logToolCompleted(ctx, "manage_schema",
		"user_id", userID,
		"approval_operation_id_hash", logx.ApprovalOperationIDHash(prepared.OperationID),
		"status", prepared.Status,
	)
	return textOnlyResult(formatSingleRowCSV([]string{
		"operation_id", "status", "approval_url", "expires_at", "summary", "assistant_instruction",
	}, payload), payload), nil
}

func toSchemaApprovalRequest(ctx context.Context, input SchemaInput) approval.SchemaMutationRequest {
	request := approval.SchemaMutationRequest{
		Base:       strings.TrimSpace(input.Base),
		Operations: make([]approval.SchemaMutationOp, 0, len(input.Operations)),
	}
	if sessionID, ok := authenticatedSessionID(ctx); ok {
		request.SessionID = sessionID
	}
	if clientID, ok := authenticatedClientID(ctx); ok {
		request.ClientID = clientID
	}
	if clientName, ok := authenticatedClientName(ctx); ok {
		request.ClientName = clientName
	}
	for _, op := range input.Operations {
		mapped := approval.SchemaMutationOp{
			Type:        strings.TrimSpace(op.Type),
			Table:       strings.TrimSpace(op.Table),
			Field:       strings.TrimSpace(op.Field),
			Name:        strings.TrimSpace(op.Name),
			Description: op.Description,
			FieldType:   strings.TrimSpace(op.FieldType),
			Options:     op.Options,
		}
		for _, field := range op.Fields {
			mapped.Fields = append(mapped.Fields, approval.SchemaFieldDefinition{
				Name:        strings.TrimSpace(field.Name),
				Type:        strings.TrimSpace(field.Type),
				Description: field.Description,
				Options:     field.Options,
			})
		}
		request.Operations = append(request.Operations, mapped)
	}
	return request
}

// unsupportedSchemaIntent recognizes attempts to delete or retype and returns
// the guidance to send back. It matches on common type names an agent might try
// even though they are not in the advertised enum, plus the case of passing a
// field_type to update_field (an attempt to change a field's type).
func unsupportedSchemaIntent(operations []SchemaOperation) (string, bool) {
	for _, operation := range operations {
		opType := strings.ToLower(strings.TrimSpace(operation.Type))
		switch {
		case strings.Contains(opType, "delete"), strings.Contains(opType, "remove"), strings.Contains(opType, "drop"):
			return schemaDeleteGuidance, true
		case strings.Contains(opType, "type") && opType != schemaOpUpdateField:
			// e.g. change_field_type, update_field_type, retype_field, convert_field_type
			return schemaRetypeGuidance, true
		case opType == schemaOpUpdateField && strings.TrimSpace(operation.FieldType) != "":
			return schemaRetypeGuidance, true
		case strings.Contains(opType, "convert"), strings.Contains(opType, "retype"):
			return schemaRetypeGuidance, true
		}
	}
	return "", false
}

func validateSchemaInput(input SchemaInput) error {
	if strings.TrimSpace(input.Base) == "" {
		return fmt.Errorf("base is required")
	}
	if len(input.Operations) == 0 {
		return fmt.Errorf("operations must contain at least one operation")
	}

	for i, operation := range input.Operations {
		opType := strings.TrimSpace(operation.Type)
		switch opType {
		case schemaOpCreateTable:
			if strings.TrimSpace(operation.Name) == "" {
				return fmt.Errorf("operations[%d].name (new table name) is required for create_table", i)
			}
			if len(operation.Fields) == 0 {
				return fmt.Errorf("operations[%d].fields must contain at least one field for create_table", i)
			}
			for j, field := range operation.Fields {
				if strings.TrimSpace(field.Name) == "" {
					return fmt.Errorf("operations[%d].fields[%d].name is required", i, j)
				}
				if strings.TrimSpace(field.Type) == "" {
					return fmt.Errorf("operations[%d].fields[%d].type is required", i, j)
				}
			}
		case schemaOpCreateField:
			if strings.TrimSpace(operation.Table) == "" {
				return fmt.Errorf("operations[%d].table is required for create_field", i)
			}
			if strings.TrimSpace(operation.Name) == "" {
				return fmt.Errorf("operations[%d].name (new field name) is required for create_field", i)
			}
			if strings.TrimSpace(operation.FieldType) == "" {
				return fmt.Errorf("operations[%d].field_type is required for create_field", i)
			}
		case schemaOpUpdateTable:
			if strings.TrimSpace(operation.Table) == "" {
				return fmt.Errorf("operations[%d].table is required for update_table", i)
			}
			if strings.TrimSpace(operation.Name) == "" && strings.TrimSpace(operation.Description) == "" {
				return fmt.Errorf("operations[%d] must set name and/or description for update_table", i)
			}
		case schemaOpUpdateField:
			if strings.TrimSpace(operation.Table) == "" {
				return fmt.Errorf("operations[%d].table is required for update_field", i)
			}
			if strings.TrimSpace(operation.Field) == "" {
				return fmt.Errorf("operations[%d].field is required for update_field", i)
			}
			if strings.TrimSpace(operation.Name) == "" && strings.TrimSpace(operation.Description) == "" {
				return fmt.Errorf("operations[%d] must set name and/or description for update_field", i)
			}
		default:
			return fmt.Errorf("operations[%d].type must be one of create_table, create_field, update_table, or update_field", i)
		}
	}

	return nil
}

func collectSchemaOperationTypes(operations []SchemaOperation) []string {
	types := make([]string, 0, len(operations))
	for _, operation := range operations {
		types = append(types, operation.Type)
	}
	return types
}
