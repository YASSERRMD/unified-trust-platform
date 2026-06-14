import { TopBar } from "@/components/layout/TopBar";
import { Table } from "@/components/ui/Table";
import { statusBadge } from "@/components/ui/Badge";
import { userApi, User } from "@/lib/api";

const DEFAULT_TENANT = process.env.DEFAULT_TENANT_ID ?? "";

async function getUsers(): Promise<User[]> {
  if (!DEFAULT_TENANT) return [];
  try {
    const res = await userApi.list(DEFAULT_TENANT);
    return res.data ?? [];
  } catch {
    return [];
  }
}

export default async function UsersPage() {
  const users = await getUsers();

  const columns = [
    { key: "email", header: "Email" },
    { key: "displayName", header: "Name" },
    {
      key: "status",
      header: "Status",
      render: (u: User) => statusBadge(u.status),
    },
    {
      key: "createdAt",
      header: "Created",
      render: (u: User) =>
        new Date(u.createdAt).toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        }),
    },
    {
      key: "id",
      header: "ID",
      render: (u: User) => (
        <span className="font-mono text-xs text-gray-400">{u.id}</span>
      ),
    },
  ];

  return (
    <>
      <TopBar
        title="Users"
        subtitle={`${users.length} user${users.length !== 1 ? "s" : ""}`}
        action={
          <button className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 transition-colors">
            Invite User
          </button>
        }
      />
      <div className="p-6">
        {!DEFAULT_TENANT && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
            Set <code className="font-mono">DEFAULT_TENANT_ID</code> environment
            variable to load users for a specific tenant.
          </div>
        )}
        <Table<User>
          columns={columns}
          rows={users}
          keyField="id"
          emptyMessage="No users found."
        />
      </div>
    </>
  );
}
