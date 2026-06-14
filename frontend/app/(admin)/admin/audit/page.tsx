import { TopBar } from "@/components/layout/TopBar";
import { Table } from "@/components/ui/Table";
import { statusBadge } from "@/components/ui/Badge";
import { auditApi, AuditEvent } from "@/lib/api";

const DEFAULT_TENANT = process.env.DEFAULT_TENANT_ID ?? "";

async function getEvents(): Promise<AuditEvent[]> {
  if (!DEFAULT_TENANT) return [];
  try {
    const res = await auditApi.list(DEFAULT_TENANT);
    return res.data ?? [];
  } catch {
    return [];
  }
}

export default async function AuditPage() {
  const events = await getEvents();

  const columns = [
    {
      key: "occurredAt",
      header: "Time",
      render: (e: AuditEvent) =>
        new Date(e.occurredAt).toLocaleString(undefined, {
          month: "short",
          day: "numeric",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        }),
    },
    { key: "action", header: "Action" },
    { key: "resource", header: "Resource" },
    {
      key: "outcome",
      header: "Outcome",
      render: (e: AuditEvent) => statusBadge(e.outcome),
    },
    {
      key: "actorEmail",
      header: "Actor",
      render: (e: AuditEvent) => e.actorEmail || e.actorId || "—",
    },
  ];

  const exportUrl = DEFAULT_TENANT
    ? `/api/v1/audit/events/export`
    : "#";

  return (
    <>
      <TopBar
        title="Audit Log"
        subtitle="Immutable record of all IAM events"
        action={
          <a
            href={exportUrl}
            className="px-4 py-2 border border-gray-300 text-gray-700 text-sm rounded-md hover:bg-gray-50 transition-colors"
          >
            Export CSV
          </a>
        }
      />
      <div className="p-6">
        {!DEFAULT_TENANT && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
            Set <code className="font-mono">DEFAULT_TENANT_ID</code> to view
            audit events.
          </div>
        )}
        <Table<AuditEvent>
          columns={columns}
          rows={events}
          keyField="id"
          emptyMessage="No audit events recorded yet."
        />
      </div>
    </>
  );
}
