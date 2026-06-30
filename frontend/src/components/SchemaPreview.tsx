import {
  FieldIcon,
  OperationBadge,
  TableLink,
  choiceColor,
  fieldTypeIcon,
  fieldTypeLabel,
  tableUrl,
} from "../formatters";
import type { SchemaFieldPreview, SchemaOperationPreview } from "../types";

// A new field name, shown with its type icon so it reads as a field and is
// clearly distinct from the surrounding verb phrase.
function FieldToken({ name, atType }: { name: string; atType?: string }) {
  return (
    <span className="schema-token is-field">
      <FieldIcon type={fieldTypeIcon(atType ?? "")} />
      {name}
    </span>
  );
}

// A concrete name being created or set (a new table, or a rename target) that
// doesn't link anywhere yet — boxed so it stands apart from the heading copy.
function NameToken({ name }: { name: string }) {
  return <span className="schema-token">{name}</span>;
}

function ChoicePills({ choices }: { choices: string[] }) {
  return (
    <span className="schema-choices">
      {choices.map((choice, index) => (
        <span
          className="at-token"
          key={`${choice}-${index}`}
          style={{ backgroundColor: choiceColor(index) }}
        >
          {choice}
        </span>
      ))}
    </span>
  );
}

function pluralFields(count: number) {
  return `${count} field${count === 1 ? "" : "s"}`;
}

// A description change shown like Airtable's revision history: the previous
// value (or "empty" when there was none) → the new value.
function DescriptionChange({ from, to }: { from?: string; to?: string }) {
  const hasFrom = !!from && from.trim().length > 0;
  return (
    <div className="schema-revision-prop">
      <span className="rev-prop-label">Description</span>
      <span className="schema-prop-diff">
        {hasFrom ? (
          <span className="rev-chip is-removed">{from}</span>
        ) : (
          <span className="rev-empty">empty</span>
        )}
        <span className="rev-arrow">→</span>
        <span className="rev-chip is-added">{to}</span>
      </span>
    </div>
  );
}

function FieldRow({ field, primary }: { field: SchemaFieldPreview; primary: boolean }) {
  const icon = fieldTypeIcon(field.type);
  return (
    <tr>
      <th>
        <FieldIcon type={icon} />
        {field.name}
        {primary ? <span className="schema-tag">Primary</span> : null}
      </th>
      <td>
        {fieldTypeLabel(field.type, icon)}
        {field.choices && field.choices.length > 0 ? <ChoicePills choices={field.choices} /> : null}
      </td>
    </tr>
  );
}

// Shared Field/Type list used by both "create table" and "add fields" cards.
function FieldList({ fields, markPrimary }: { fields: SchemaFieldPreview[]; markPrimary: boolean }) {
  return (
    <div className="table-shell">
      <table>
        <thead>
          <tr>
            <th>Field</th>
            <th>Type</th>
          </tr>
        </thead>
        <tbody>
          {fields.map((field, index) => (
            <FieldRow key={`${field.name}-${index}`} field={field} primary={markPrimary && index === 0} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

// One card for all fields added to the same table, so adding several columns
// reads as a single change rather than several near-identical cards.
function AddFieldsCard({
  tableId,
  tableName,
  fields,
  baseId,
}: {
  tableId?: string;
  tableName?: string;
  fields: SchemaFieldPreview[];
  baseId?: string;
}) {
  return (
    <section className="preview-card">
      <CardHeader kind="create">
        Add {pluralFields(fields.length)} to <TableLink href={tableUrl(baseId, tableId)} name={tableName ?? ""} />
      </CardHeader>
      <FieldList fields={fields} markPrimary={false} />
    </section>
  );
}

// One card for all field renames/description edits on the same table.
function UpdateFieldsCard({
  tableId,
  tableName,
  ops,
  baseId,
}: {
  tableId?: string;
  tableName?: string;
  ops: SchemaOperationPreview[];
  baseId?: string;
}) {
  const allRenames = ops.every((op) => op.new_name);
  const verb = allRenames ? "Rename" : "Update";
  return (
    <section className="preview-card">
      <CardHeader kind="update">
        {verb} {pluralFields(ops.length)} in <TableLink href={tableUrl(baseId, tableId)} name={tableName ?? ""} />
      </CardHeader>
      <div className="schema-revisions">
        {ops.map((op, index) => (
          <div className="schema-revision" key={`${op.field_name}-${index}`}>
            <div className="schema-revision-name">
              {op.new_name ? (
                <>
                  <span className="rev-chip is-removed">{op.field_name}</span>
                  <span className="rev-arrow">→</span>
                  <span className="rev-chip is-added">{op.new_name}</span>
                </>
              ) : (
                <span className="rev-field">{op.field_name}</span>
              )}
            </div>
            {op.description ? <DescriptionChange from={op.old_description} to={op.description} /> : null}
          </div>
        ))}
      </div>
    </section>
  );
}

function CardHeader({ kind, children }: { kind: "create" | "update"; children: React.ReactNode }) {
  return (
    <div className="preview-header">
      <div className="preview-title">
        <OperationBadge kind={kind} />
        <h2>{children}</h2>
      </div>
    </div>
  );
}

// A single operation that isn't part of a multi-field group: create a table,
// add one field, or rename/describe one table or field.
export function SchemaPreview({ operation, baseId }: { operation: SchemaOperationPreview; baseId?: string }) {
  const isCreate = operation.type === "create_table" || operation.type === "create_field";
  return (
    <section className="preview-card">
      <CardHeader kind={isCreate ? "create" : "update"}>{schemaHeading(operation, baseId)}</CardHeader>
      <SchemaBody operation={operation} />
    </section>
  );
}

function schemaHeading(operation: SchemaOperationPreview, baseId?: string) {
  const tableLink = <TableLink href={tableUrl(baseId, operation.table_id)} name={operation.table_name ?? ""} />;

  switch (operation.type) {
    case "create_table":
      return (
        <>
          Create table <NameToken name={operation.table_name ?? "Untitled"} />
        </>
      );
    case "create_field":
      return (
        <>
          Add field <FieldToken name={operation.field_name ?? "Untitled"} atType={operation.field_type} /> to{" "}
          {tableLink}
        </>
      );
    case "update_table":
      return operation.new_name ? (
        <>
          Rename table {tableLink} to <NameToken name={operation.new_name} />
        </>
      ) : (
        <>Update table {tableLink}</>
      );
    case "update_field":
      return operation.new_name ? (
        <>
          Rename field <NameToken name={operation.field_name ?? "field"} /> to{" "}
          <NameToken name={operation.new_name} /> in {tableLink}
        </>
      ) : (
        <>
          Update field <NameToken name={operation.field_name ?? "field"} /> in {tableLink}
        </>
      );
    default:
      return <>Change schema</>;
  }
}

function SchemaBody({ operation }: { operation: SchemaOperationPreview }) {
  if (operation.type === "create_table") {
    return <FieldList fields={operation.fields ?? []} markPrimary={true} />;
  }

  if (operation.type === "create_field") {
    const icon = fieldTypeIcon(operation.field_type ?? "");
    return (
      <dl className="schema-card">
        <div className="schema-row">
          <dt>Type</dt>
          <dd className="schema-type">
            <FieldIcon type={icon} />
            {fieldTypeLabel(operation.field_type, icon)}
          </dd>
        </div>
        {operation.choices && operation.choices.length > 0 ? (
          <div className="schema-row">
            <dt>Options</dt>
            <dd>
              <ChoicePills choices={operation.choices} />
            </dd>
          </div>
        ) : null}
        {operation.description ? (
          <div className="schema-row">
            <dt>Description</dt>
            <dd>{operation.description}</dd>
          </div>
        ) : null}
      </dl>
    );
  }

  // update_table / update_field: the rename itself is in the heading; show a
  // description change (old -> new) when one is set.
  if (operation.description) {
    return (
      <div className="schema-revisions">
        <div className="schema-revision">
          <DescriptionChange from={operation.old_description} to={operation.description} />
        </div>
      </div>
    );
  }
  return null;
}

type SchemaRenderItem =
  | { kind: "op"; key: string; op: SchemaOperationPreview }
  | { kind: "add_fields"; key: string; tableId?: string; tableName?: string; fields: SchemaFieldPreview[] }
  | { kind: "update_fields"; key: string; tableId?: string; tableName?: string; ops: SchemaOperationPreview[] };

function tableKey(op: SchemaOperationPreview) {
  return op.table_id ?? op.table_name ?? "";
}

// Collapses create_field and update_field operations that share a table into a
// single card each (anchored at the table's first such op), but only when there
// are 2+ — a lone add/rename keeps its richer individual card. Every other
// operation stays its own card in place.
function groupSchemaOperations(operations: SchemaOperationPreview[]): SchemaRenderItem[] {
  const createCounts = new Map<string, number>();
  const updateCounts = new Map<string, number>();
  for (const op of operations) {
    const key = tableKey(op);
    if (op.type === "create_field") createCounts.set(key, (createCounts.get(key) ?? 0) + 1);
    else if (op.type === "update_field") updateCounts.set(key, (updateCounts.get(key) ?? 0) + 1);
  }

  const items: SchemaRenderItem[] = [];
  const addIndex = new Map<string, number>();
  const updIndex = new Map<string, number>();

  operations.forEach((op, i) => {
    const key = tableKey(op);

    if (op.type === "create_field" && (createCounts.get(key) ?? 0) >= 2) {
      const field: SchemaFieldPreview = {
        name: op.field_name ?? "Untitled",
        type: op.field_type ?? "",
        choices: op.choices,
      };
      const at = addIndex.get(key);
      if (at !== undefined) {
        (items[at] as Extract<SchemaRenderItem, { kind: "add_fields" }>).fields.push(field);
        return;
      }
      addIndex.set(key, items.length);
      items.push({ kind: "add_fields", key: `add-${key}-${i}`, tableId: op.table_id, tableName: op.table_name, fields: [field] });
      return;
    }

    if (op.type === "update_field" && (updateCounts.get(key) ?? 0) >= 2) {
      const at = updIndex.get(key);
      if (at !== undefined) {
        (items[at] as Extract<SchemaRenderItem, { kind: "update_fields" }>).ops.push(op);
        return;
      }
      updIndex.set(key, items.length);
      items.push({ kind: "update_fields", key: `upd-${key}-${i}`, tableId: op.table_id, tableName: op.table_name, ops: [op] });
      return;
    }

    items.push({ kind: "op", key: `op-${op.type}-${i}`, op });
  });

  return items;
}

export function SchemaOperationList({
  operations,
  baseId,
}: {
  operations: SchemaOperationPreview[];
  baseId?: string;
}) {
  return (
    <>
      {groupSchemaOperations(operations).map((item) => {
        if (item.kind === "add_fields") {
          return (
            <AddFieldsCard
              key={item.key}
              tableId={item.tableId}
              tableName={item.tableName}
              fields={item.fields}
              baseId={baseId}
            />
          );
        }
        if (item.kind === "update_fields") {
          return (
            <UpdateFieldsCard
              key={item.key}
              tableId={item.tableId}
              tableName={item.tableName}
              ops={item.ops}
              baseId={baseId}
            />
          );
        }
        return <SchemaPreview key={item.key} operation={item.op} baseId={baseId} />;
      })}
    </>
  );
}
