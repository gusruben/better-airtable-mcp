import {
  FieldIcon,
  OperationBadge,
  TableLink,
  buildColumns,
  fieldTypeLabel,
  formatFieldValue,
  recordCountLabel,
  tableUrl,
} from "../formatters";
import type { LinkedRecordRef, OperationPreview } from "../types";

interface RecordTableProps {
  operation: OperationPreview;
  baseId?: string;
  linked?: Record<string, LinkedRecordRef>;
}

export function RecordTable({ operation, baseId, linked }: RecordTableProps) {
  const columns = buildColumns(operation.fields, operation.records);
  const tableName = operation.original_table_name ?? operation.table;
  const href = tableUrl(baseId, operation.table_id);
  const ctx = { baseId, linked };

  return (
    <section className="preview-card">
      <div className="preview-header">
        <div className="preview-title">
          <OperationBadge kind="create" />
          <h2>
            Create {recordCountLabel(operation.records.length)} in{" "}
            <TableLink href={href} name={tableName} />
          </h2>
        </div>
      </div>
      <div className="table-shell">
        <table>
          <thead>
            <tr>
              {columns.map((column, i) => (
                <th key={i} data-tooltip={fieldTypeLabel(column.atType, column.type)}>
                  <FieldIcon type={column.type} />
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {operation.records.map((record, index) => (
              <tr key={`${record.id ?? "new"}-${index}`}>
                {columns.map((column, i) => {
                  const value = column.get(record.fields);
                  const content = formatFieldValue(value, ctx);
                  return i === 0 ? (
                    <th key={i}>{content}</th>
                  ) : (
                    <td key={i} className={typeof value === "number" ? "is-num" : undefined}>
                      {content}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
