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
import type { LinkedRecordRef, OperationPreview } from "../types";

interface DeletePreviewProps {
  operation: OperationPreview;
  baseId?: string;
  linked?: Record<string, LinkedRecordRef>;
}

export function DeletePreview({ operation, baseId, linked }: DeletePreviewProps) {
  const tableName = operation.original_table_name ?? operation.table;
  const href = tableUrl(baseId, operation.table_id);
  const ctx = { baseId, linked };

  return (
    <section className="preview-card">
      <div className="preview-header">
        <div className="preview-title">
          <OperationBadge kind="delete" />
          <h2>
            Delete {recordCountLabel(operation.records.length)} from{" "}
            <TableLink href={href} name={tableName} />
          </h2>
        </div>
      </div>
      <div className="record-grid">
        {operation.records.map((record, index) => {
          const current = record.current_fields ?? {};
          const ordered = orderFieldKeys(Object.keys(current), operation.fields, (key) => current[key]);
          const primary = primaryFieldDisplay(record, operation.fields);
          return (
            <div className="record-card" key={`${record.id ?? "delete"}-${index}`}>
              <div className="record-title">
                <RecordNameLink baseId={baseId} tableId={operation.table_id} recordId={record.id} primary={primary} />
                {record.id ? <CopyId id={record.id} /> : null}
              </div>
              <div className="table-shell">
                <table>
                  <thead>
                    <tr>
                      <th>Field</th>
                      <th>Current value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ordered.map(({ key, label, type, atType }) => {
                      const value = current[key];
                      return (
                        <tr key={key}>
                          <th data-tooltip={fieldTypeLabel(atType, type)}>
                            <FieldIcon type={type} />
                            {label}
                          </th>
                          <td className={typeof value === "number" ? "is-num" : undefined}>
                            {formatFieldValue(value, ctx)}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
