import { TopBar } from "@/components/layout/TopBar";
import { Table } from "@/components/ui/Table";
import { Badge } from "@/components/ui/Badge";
import { roleApi, Role } from "@/lib/api";

const DEFAULT_TENANT = process.env.DEFAULT_TENANT_ID ?? "";

async function getRoles(): Promise<Role[]> {
  if (!DEFAULT_TENANT) return [];
  try {
    const res = await roleApi.list(DEFAULT_TENANT);
    return res.data ?? [];
  } catch {
    return [];
  }
}

export default async function RolesPage() {
  const roles = await getRoles();

  const columns = [
    { key: "name", header: "Role Name" },
    { key: "description", header: "Description" },
    {
      key: "isSystem",
      header: "Type",
      render: (r: Role) =>
        r.isSystem ? (
          <Badge label="System" variant="blue" />
        ) : (
          <Badge label="Custom" variant="gray" />
        ),
    },
    {
      key: "createdAt",
      header: "Created",
      render: (r: Role) =>
        new Date(r.createdAt).toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        }),
    },
  ];

  return (
    <>
      <TopBar
        title="Roles & Permissions"
        subtitle={`${roles.length} role${roles.length !== 1 ? "s" : ""}`}
        action={
          <button className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors">
            New Role
          </button>
        }
      />
      <div className="p-6">
        {!DEFAULT_TENANT && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
            Set <code className="font-mono">DEFAULT_TENANT_ID</code> to load
            roles.
          </div>
        )}
        <Table<Role>
          columns={columns}
          rows={roles}
          keyField="id"
          emptyMessage="No roles found."
        />
      </div>
    </>
  );
}
