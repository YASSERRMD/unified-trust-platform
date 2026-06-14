import { TopBar } from "@/components/layout/TopBar";
import { Table } from "@/components/ui/Table";
import { statusBadge, Badge } from "@/components/ui/Badge";
import { policyApi, Policy } from "@/lib/api";

const DEFAULT_TENANT = process.env.DEFAULT_TENANT_ID ?? "";

async function getPolicies(): Promise<Policy[]> {
  if (!DEFAULT_TENANT) return [];
  try {
    const res = await policyApi.list(DEFAULT_TENANT);
    return res.data ?? [];
  } catch {
    return [];
  }
}

export default async function PoliciesPage() {
  const policies = await getPolicies();

  const columns = [
    { key: "name", header: "Policy Name" },
    {
      key: "effect",
      header: "Effect",
      render: (p: Policy) => statusBadge(p.effect),
    },
    { key: "priority", header: "Priority" },
    {
      key: "actions",
      header: "Actions",
      render: (p: Policy) => (
        <span className="font-mono text-xs">
          {(p.actions ?? []).join(", ") || "—"}
        </span>
      ),
    },
    {
      key: "isActive",
      header: "Active",
      render: (p: Policy) =>
        p.isActive ? (
          <Badge label="Active" variant="green" />
        ) : (
          <Badge label="Inactive" variant="gray" />
        ),
    },
  ];

  return (
    <>
      <TopBar
        title="Authorization Policies"
        subtitle={`${policies.length} polic${policies.length !== 1 ? "ies" : "y"}`}
        action={
          <button className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors">
            New Policy
          </button>
        }
      />
      <div className="p-6">
        {!DEFAULT_TENANT && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
            Set <code className="font-mono">DEFAULT_TENANT_ID</code> to load
            policies.
          </div>
        )}
        <Table<Policy>
          columns={columns}
          rows={policies}
          keyField="id"
          emptyMessage="No policies defined."
        />
      </div>
    </>
  );
}
