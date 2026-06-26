import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { DeletePreview } from "./components/DeletePreview";
import { DiffView } from "./components/DiffView";
import { McpDebugPage } from "./McpDebugPage";
import { RecordTable } from "./components/RecordTable";
import {
  BaseLink,
  absoluteTime,
  countdownLabel,
  getOperationIDFromPath,
  relativeTime,
  statusLabel,
  statusTone,
} from "./formatters";
import type {
  ExecutionResult,
  LinkedRecordRef,
  OperationPreview,
  OperationView,
} from "./types";
import "./styles.css";

interface AppProps {
  pathname?: string;
  fetchImpl?: typeof fetch;
}

function isDebugPath(pathname: string): boolean {
  return pathname === "/debug" || pathname === "/debug/";
}

function operationClientLabel(operation: OperationView): string {
  if (operation.mcp_client_name && operation.mcp_client_id) {
    return `${operation.mcp_client_name} (${operation.mcp_client_id})`;
  }
  return operation.mcp_client_name ?? operation.mcp_client_id ?? "Unknown client";
}

function useNow(intervalMs = 1000): Date {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), intervalMs);
    return () => window.clearInterval(timer);
  }, [intervalMs]);
  return now;
}

// Airtable-style custom tooltip. A single delegated layer reads `data-tooltip`
// from any hovered/focused element and renders one portal tooltip on the body,
// so it escapes the grid's horizontal-scroll clipping (the reason Airtable
// portals its own tooltips). Styled to match Airtable's `.ui-tooltip`.
interface TipState {
  text: string;
  left: number;
  top: number;
  placement: "top" | "bottom";
}

function TooltipLayer() {
  const [tip, setTip] = useState<TipState | null>(null);
  useEffect(() => {
    let target: Element | null = null;
    let timer: number | undefined;

    const reveal = (el: Element) => {
      const text = el.getAttribute("data-tooltip");
      if (!text) return;
      target = el;
      window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        const rect = el.getBoundingClientRect();
        const placement = rect.top > 44 ? "top" : "bottom";
        setTip({
          text,
          left: Math.round(rect.left + rect.width / 2),
          top: Math.round(placement === "top" ? rect.top : rect.bottom),
          placement,
        });
      }, 250);
    };
    const dismiss = () => {
      target = null;
      window.clearTimeout(timer);
      setTip(null);
    };
    const onOver = (event: Event) => {
      const el = (event.target as Element | null)?.closest?.("[data-tooltip]");
      if (el && el !== target) reveal(el);
    };
    const onOut = (event: MouseEvent) => {
      if (!target) return;
      const related = event.relatedTarget as Node | null;
      if (!related || !target.contains(related)) dismiss();
    };
    const onFocusIn = (event: Event) => {
      const el = (event.target as Element | null)?.closest?.("[data-tooltip]");
      if (el && el !== target) reveal(el);
    };

    document.addEventListener("mouseover", onOver, true);
    document.addEventListener("mouseout", onOut as EventListener, true);
    document.addEventListener("focusin", onFocusIn, true);
    document.addEventListener("focusout", dismiss, true);
    window.addEventListener("scroll", dismiss, true);
    return () => {
      document.removeEventListener("mouseover", onOver, true);
      document.removeEventListener("mouseout", onOut as EventListener, true);
      document.removeEventListener("focusin", onFocusIn, true);
      document.removeEventListener("focusout", dismiss, true);
      window.removeEventListener("scroll", dismiss, true);
      window.clearTimeout(timer);
    };
  }, []);

  if (!tip) return null;
  return createPortal(
    <div
      className={`at-tooltip is-${tip.placement}`}
      style={{ left: tip.left, top: tip.top }}
      role="tooltip"
    >
      {tip.text}
    </div>,
    document.body,
  );
}

function MetaItem({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <span className="meta-item">
      <span className="meta-key">{label}</span>
      <span className="meta-val">{children}</span>
    </span>
  );
}

function ResultPanel({ result }: { result: ExecutionResult }) {
  const stats = [
    { label: "Created", value: result.created_record_ids?.length ?? 0, tone: "created" },
    { label: "Updated", value: result.updated_record_ids?.length ?? 0, tone: "updated" },
    { label: "Deleted", value: result.deleted_record_ids?.length ?? 0, tone: "deleted" },
    { label: "Batches", value: result.completed_batches, tone: "neutral" },
  ];

  return (
    <section className="preview-card">
      <h2>Result</h2>
      <div className="result-grid">
        {stats.map((stat) => {
          const toned = stat.tone !== "neutral" && stat.value > 0;
          return (
            <div className="result-stat" key={stat.label}>
              <div className={`stat-value${toned ? ` is-${stat.tone}` : ""}`}>
                {stat.value.toLocaleString()}
              </div>
              <div className="stat-label">{stat.label}</div>
            </div>
          );
        })}
      </div>
      {typeof result.failed_batch === "number" ? (
        <p className="error-text">Batch {result.failed_batch + 1} did not complete.</p>
      ) : null}
    </section>
  );
}

function OperationSection({
  operation,
  baseId,
  linked,
}: {
  operation: OperationPreview;
  baseId?: string;
  linked?: Record<string, LinkedRecordRef>;
}) {
  if (operation.type === "create_records") {
    return <RecordTable operation={operation} baseId={baseId} linked={linked} />;
  }
  if (operation.type === "delete_records") {
    return <DeletePreview operation={operation} baseId={baseId} linked={linked} />;
  }
  return <DiffView operation={operation} baseId={baseId} linked={linked} />;
}

function ApprovalView({
  operation,
  onAction,
  busyAction,
}: {
  operation: OperationView;
  onAction: (action: "approve" | "reject") => Promise<void>;
  busyAction: "approve" | "reject" | null;
}) {
  const busy = busyAction !== null;
  const now = useNow();
  const tone = statusTone(operation.status);
  const isPending = operation.status === "pending_approval";
  const remaining = isPending ? countdownLabel(operation.expires_at, now) : "";
  const canAct = operation.can_approve || operation.can_reject;
  const hasOperations = operation.operations.length > 0;
  const syncedAt = new Date(operation.last_synced_at);
  const syncedValid = !Number.isNaN(syncedAt.getTime()) && syncedAt.getFullYear() > 2000;

  return (
    <main className="page-shell">
      <section className="hero-card">
        <div className={`status-banner is-${tone}`}>
          <span className="status-dot" />
          <span>
            {statusLabel(operation.status)}
            {isPending && remaining !== "expired" ? ` · ${remaining} left` : ""}
          </span>
        </div>

        <h1>{operation.summary}</h1>

        <div className="meta-row">
          <MetaItem label="Base">
            <BaseLink baseId={operation.base_id} name={operation.base_name} />
          </MetaItem>
          <MetaItem label="Requested by">{operationClientLabel(operation)}</MetaItem>
          <MetaItem label="Requested">
            <time dateTime={operation.created_at} data-tooltip={absoluteTime(operation.created_at)}>
              {relativeTime(operation.created_at, now)}
            </time>
          </MetaItem>
          {isPending ? (
            <MetaItem label="Expires">{`${countdownLabel(
              operation.expires_at,
              now,
            )}`}</MetaItem>
          ) : null}
        </div>

        <div className="notice-stack">
          <p className="notice">
            Anyone with this link can approve or reject the request until it expires.
          </p>
          <p className="notice">
            {syncedValid ? (
              <>
                This is a snapshot of Airtable data taken{" "}
                <time dateTime={operation.last_synced_at} data-tooltip={absoluteTime(operation.last_synced_at)}>
                  {relativeTime(operation.last_synced_at, now)} ({absoluteTime(operation.last_synced_at)})
                </time>
                , so live records may have changed since.
              </>
            ) : (
              "This is a snapshot of Airtable data, so live records may have changed since."
            )}
          </p>
        </div>

        {operation.error ? (
          <div className="status-banner is-danger" style={{ marginTop: "16px" }}>
            <span className="status-dot" />
            <span>{operation.error}</span>
          </div>
        ) : null}

        {canAct ? (
          <div className="action-row">
            <button
              className="action-button reject"
              disabled={!operation.can_reject || busy}
              onClick={() => void onAction("reject")}
            >
              {busyAction === "reject" ? "Rejecting…" : "Reject"}
            </button>
            <button
              className="action-button approve"
              disabled={!operation.can_approve || busy}
              onClick={() => void onAction("approve")}
            >
              {busyAction === "approve" ? "Approving…" : "Approve"}
            </button>
          </div>
        ) : null}
      </section>

      {operation.result ? <ResultPanel result={operation.result} /> : null}

      {hasOperations ? (
        <section className="preview-stack">
          {operation.operations.map((item, index) => (
            <OperationSection
              key={`${item.type}-${item.table}-${index}`}
              operation={item}
              baseId={operation.base_id}
              linked={operation.linked_records}
            />
          ))}
        </section>
      ) : (
        <section className="preview-card">
          <p className="meta-text">This request doesn't change any records.</p>
        </section>
      )}
    </main>
  );
}

function MessageCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <main className="page-shell">
      <section className="hero-card">
        <h1>{title}</h1>
        {children}
      </section>
    </main>
  );
}

export default function App({
  pathname = window.location.pathname,
  fetchImpl = window.fetch.bind(window),
}: AppProps) {
  if (isDebugPath(pathname)) {
    return (
      <>
        <McpDebugPage fetchImpl={fetchImpl} />
        <TooltipLayer />
      </>
    );
  }

  const operationID = useMemo(() => getOperationIDFromPath(pathname), [pathname]);
  const [operation, setOperation] = useState<OperationView | null>(null);
  const [error, setError] = useState("");
  const [busyAction, setBusyAction] = useState<"approve" | "reject" | null>(null);

  useEffect(() => {
    if (!operationID) {
      setError("This approval link is missing an operation ID.");
      return;
    }

    let cancelled = false;
    async function load() {
      const response = await fetchImpl(`/api/operations/${operationID}`, {
        headers: { Accept: "application/json" },
      });
      const payload = (await response.json()) as OperationView & { error?: string };
      if (cancelled) {
        return;
      }
      if (!response.ok) {
        setError(payload.error ?? "We couldn't load this approval request.");
        return;
      }
      setOperation(payload);
      setError("");
    }

    void load().catch((cause) => {
      if (!cancelled) {
        setError(String(cause));
      }
    });

    return () => {
      cancelled = true;
    };
  }, [fetchImpl, operationID]);

  async function handleAction(action: "approve" | "reject") {
    if (!operationID) {
      return;
    }
    setBusyAction(action);
    try {
      const response = await fetchImpl(`/api/operations/${operationID}/${action}`, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({}),
      });
      const payload = (await response.json()) as OperationView & { error?: string };
      if (!response.ok) {
        setError(payload.error ?? `We couldn't ${action} this request.`);
        return;
      }
      setOperation(payload);
      setError("");
    } catch (cause) {
      setError(String(cause));
    } finally {
      setBusyAction(null);
    }
  }

  if (error) {
    return (
      <>
        <MessageCard title="Approval request">
          <div className="status-banner is-danger">
            <span className="status-dot" />
            <span>{error}</span>
          </div>
        </MessageCard>
        <TooltipLayer />
      </>
    );
  }

  if (!operation) {
    return (
      <>
        <MessageCard title="Approval request">
          <p className="meta-text">Loading the request…</p>
        </MessageCard>
        <TooltipLayer />
      </>
    );
  }

  return (
    <>
      <ApprovalView operation={operation} onAction={handleAction} busyAction={busyAction} />
      <TooltipLayer />
    </>
  );
}
