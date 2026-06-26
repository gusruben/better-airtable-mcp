import {
  CopyId,
  FieldIcon,
  OperationBadge,
  RecordNameLink,
  TableLink,
  fieldTypeLabel,
  formatFieldValue,
  orderFieldKeys,
  primaryFieldDisplay,
  recordCountLabel,
  tableUrl,
} from "../formatters";
import type {
  FieldMeta,
  LinkedRecordRef,
  OperationPreview,
  OperationPreviewRecord,
} from "../types";

interface DiffViewProps {
  operation: OperationPreview;
  baseId?: string;
  linked?: Record<string, LinkedRecordRef>;
}

function DiffRecord({
  record,
  fields,
  baseId,
  tableId,
  linked,
}: {
  record: OperationPreviewRecord;
  fields?: FieldMeta[];
  baseId?: string;
  tableId?: string;
  linked?: Record<string, LinkedRecordRef>;
}) {
  const currentFields = record.current_fields ?? {};
  const nextFields = record.fields ?? {};
  const ctx = { baseId, linked };

  const requestedKeys = Object.keys(nextFields);
  const changedKeys = requestedKeys.filter(
    (key) => JSON.stringify(currentFields[key]) !== JSON.stringify(nextFields[key]),
  );
  const changed = orderFieldKeys(changedKeys, fields, (key) => nextFields[key]);

  const unchangedNames = [
    ...requestedKeys.filter((key) => !changedKeys.includes(key)),
    ...Object.keys(currentFields).filter((key) => !(key in nextFields)),
  ].sort();
  const tooltipText =
    unchangedNames.slice(0, 12).join(", ") +
    (unchangedNames.length > 12 ? `, and ${unchangedNames.length - 12} more` : "");

  const primary = primaryFieldDisplay(record, fields);

  return (
    <div className="record-card">
      <div className="record-title">
        <RecordNameLink baseId={baseId} tableId={tableId} recordId={record.id} primary={primary} />
        {record.id ? <CopyId id={record.id} /> : null}
      </div>
      {changed.length > 0 ? (
        <div className="table-shell">
          <table className="diff-table">
            <thead>
              <tr>
                <th>Field</th>
                <th>Was</th>
                <th>Now</th>
              </tr>
            </thead>
            <tbody>
              {changed.map(({ key, label, type, atType }) => {
                const before = currentFields[key];
                const after = nextFields[key];
                return (
                  <tr key={key}>
                    <th data-tooltip={fieldTypeLabel(atType, type)}>
                      <FieldIcon type={type} />
                      {label}
                    </th>
                    <td className="diff-was">{formatFieldValue(before, ctx)}</td>
                    <td className="diff-now">{formatFieldValue(after, ctx)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="unchanged-note">No field values change for this record.</p>
      )}
      {unchangedNames.length > 0 && (
        <p className="unchanged-note">
          <span className="unchanged-hint" data-tooltip={tooltipText}>
            {unchangedNames.length} other field{unchangedNames.length === 1 ? "" : "s"}
          </span>{" "}
          unchanged
        </p>
      )}
    </div>
  );
}

export function DiffView({ operation, baseId, linked }: DiffViewProps) {
  const tableName = operation.original_table_name ?? operation.table;
  const href = tableUrl(baseId, operation.table_id);

  return (
    <section className="preview-card">
      <div className="preview-header">
        <div className="preview-title">
          <OperationBadge kind="update" />
          <h2>
            Update {recordCountLabel(operation.records.length)} in{" "}
            <TableLink href={href} name={tableName} />
          </h2>
        </div>
      </div>
      <div className="record-grid">
        {operation.records.map((record, index) => (
          <DiffRecord
            key={`${record.id ?? "record"}-${index}`}
            record={record}
            fields={operation.fields}
            baseId={baseId}
            tableId={operation.table_id}
            linked={linked}
          />
        ))}
      </div>
    </section>
  );
}
