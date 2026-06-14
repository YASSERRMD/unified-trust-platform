import { TopBar } from "@/components/layout/TopBar";
import { Table } from "@/components/ui/Table";
import { statusBadge } from "@/components/ui/Badge";
import { tenantApi, Tenant } from "@/lib/api";

async function getTenants(): Promise<Tenant[]> {
  try {
    const res = await tenantApi.list();
    return res.data ?? [];
  } catch {
    return [];
  }
}

export default async function TenantsPage() {
  const tenants = await getTenants();

  const columns = [
    { key: "name", header: "Name" },
    { key: "slug", header: "Slug" },
    {
      key: "status",
      header: "Status",
      render: (t: Tenant) => statusBadge(t.status),
    },
    {
      key: "createdAt",
      header: "Created",
      render: (t: Tenant) =>
        new Date(t.createdAt).toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        }),
    },
    {
      key: "id",
      header: "ID",
      render: (t: Tenant) => (
        <span className="font-mono text-xs text-gray-400">{t.id}</span>
      ),
    },
  ];

  return (
    <>
      <TopBar
        title="Tenants"
        subtitle={`${tenants.length} tenant${tenants.length !== 1 ? "s" : ""}`}
        action={
          <button className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors">
            New Tenant
          </button>
        }
      />
      <div className="p-6">
        <Table<Tenant>
          columns={columns}
          rows={tenants}
          keyField="id"
          emptyMessage="No tenants yet. Create the first one."
        />
      </div>
    </>
  );
}
