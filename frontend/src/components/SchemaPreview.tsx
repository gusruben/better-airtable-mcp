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

interface SchemaPreviewProps {
  operation: SchemaOperationPreview;
  baseId?: string;
}

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

// Renders one schema operation (create/rename a table or field) in the same card
// idiom as the record previews, reusing Airtable's field-type icons and labels.
export function SchemaPreview({ operation, baseId }: SchemaPreviewProps) {
  const isCreate = operation.type === "create_table" || operation.type === "create_field";

  return (
    <section className="preview-card">
      <div className="preview-header">
        <div className="preview-title">
          <OperationBadge kind={isCreate ? "create" : "update"} />
          <h2>{schemaHeading(operation, baseId)}</h2>
        </div>
      </div>
      <SchemaBody operation={operation} />
    </section>
  );
}

function schemaHeading(operation: SchemaOperationPreview, baseId?: string) {
  const tableHref = tableUrl(baseId, operation.table_id);
  const tableLink = <TableLink href={tableHref} name={operation.table_name ?? ""} />;

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
    const fields = operation.fields ?? [];
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
              <CreateTableFieldRow key={`${field.name}-${index}`} field={field} primary={index === 0} />
            ))}
          </tbody>
        </table>
      </div>
    );
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
  // description change when one is set.
  if (operation.description) {
    return (
      <dl className="schema-card">
        <div className="schema-row">
          <dt>Description</dt>
          <dd>{operation.description}</dd>
        </div>
      </dl>
    );
  }
  return null;
}

function CreateTableFieldRow({ field, primary }: { field: SchemaFieldPreview; primary: boolean }) {
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
