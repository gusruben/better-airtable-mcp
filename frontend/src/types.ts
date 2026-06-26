export type OperationStatus =
  | "pending_approval"
  | "approved"
  | "rejected"
  | "expired"
  | "executing"
  | "completed"
  | "partially_completed"
  | "failed";

export interface OperationPreviewRecord {
  id?: string;
  fields?: Record<string, unknown>;
  current_fields?: Record<string, unknown>;
}

export interface FieldMeta {
  name: string;
  key?: string;
  type: string;
}

export interface LinkedRecordRef {
  name?: string;
  table_id?: string;
}

export interface OperationPreview {
  type: "create_records" | "update_records" | "delete_records";
  table: string;
  original_table_name?: string;
  table_id?: string;
  fields?: FieldMeta[];
  records: OperationPreviewRecord[];
}

export interface ExecutionResult {
  created_record_ids?: string[];
  updated_record_ids?: string[];
  deleted_record_ids?: string[];
  completed_batches: number;
  failed_batch?: number;
}

export type SchemaOperationType =
  | "create_table"
  | "create_field"
  | "update_table"
  | "update_field";

export interface SchemaFieldPreview {
  name: string;
  type: string;
  description?: string;
  choices?: string[];
}

export interface SchemaOperationPreview {
  type: SchemaOperationType;
  table_id?: string;
  table_name?: string;
  field_id?: string;
  field_name?: string;
  new_name?: string;
  description?: string;
  field_type?: string;
  choices?: string[];
  fields?: SchemaFieldPreview[];
}

export interface SchemaExecutionResult {
  completed_operations: number;
  failed_operation?: number;
  created_table_ids?: string[];
  created_field_ids?: string[];
  updated_table_ids?: string[];
  updated_field_ids?: string[];
}

export interface OperationView {
  operation_id: string;
  status: OperationStatus;
  approval_url: string;
  base_id: string;
  base_name: string;
  mcp_session_id?: string;
  mcp_client_id?: string;
  mcp_client_name?: string;
  summary: string;
  created_at: string;
  expires_at: string;
  resolved_at?: string;
  last_synced_at: string;
  operation_type?: "record_mutation" | "schema_mutation";
  operations: OperationPreview[];
  linked_records?: Record<string, LinkedRecordRef>;
  schema_operations?: SchemaOperationPreview[];
  result?: ExecutionResult;
  schema_result?: SchemaExecutionResult;
  error?: string;
  approval_url_is_credential: boolean;
  preview_is_snapshot: boolean;
  can_approve: boolean;
  can_reject: boolean;
}
