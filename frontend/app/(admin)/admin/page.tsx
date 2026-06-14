import { TopBar } from "@/components/layout/TopBar";

const stats = [
  { label: "Tenants", value: "—", href: "/admin/tenants" },
  { label: "Users", value: "—", href: "/admin/users" },
  { label: "Active Policies", value: "—", href: "/admin/policies" },
  { label: "Audit Events (24h)", value: "—", href: "/admin/audit" },
];

export default function DashboardPage() {
  return (
    <>
      <TopBar
        title="Dashboard"
        subtitle="Unified Trust Platform — Enterprise IAM"
      />
      <div className="p-6 flex-1">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          {stats.map((s) => (
            <a
              key={s.label}
              href={s.href}
              className="bg-white rounded-lg border border-gray-200 p-5 hover:shadow-md transition-shadow"
            >
              <p className="text-sm text-gray-500">{s.label}</p>
              <p className="text-3xl font-bold text-gray-900 mt-1">{s.value}</p>
            </a>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-white rounded-lg border border-gray-200 p-5">
            <h3 className="font-semibold text-gray-900 mb-3">Quick Links</h3>
            <ul className="space-y-2 text-sm">
              <li>
                <a
                  href="/admin/tenants"
                  className="text-blue-600 hover:underline"
                >
                  Manage Tenants
                </a>
              </li>
              <li>
                <a href="/admin/users" className="text-blue-600 hover:underline">
                  Manage Users
                </a>
              </li>
              <li>
                <a href="/admin/roles" className="text-blue-600 hover:underline">
                  Roles & Permissions
                </a>
              </li>
              <li>
                <a
                  href="/admin/policies"
                  className="text-blue-600 hover:underline"
                >
                  Authorization Policies
                </a>
              </li>
              <li>
                <a href="/admin/audit" className="text-blue-600 hover:underline">
                  Audit Events
                </a>
              </li>
            </ul>
          </div>

          <div className="bg-white rounded-lg border border-gray-200 p-5">
            <h3 className="font-semibold text-gray-900 mb-3">System Status</h3>
            <ul className="space-y-2 text-sm">
              <StatusRow label="Backend API" status="unknown" />
              <StatusRow label="Database" status="unknown" />
              <StatusRow label="Redis Cache" status="unknown" />
              <StatusRow label="OIDC / OAuth2" status="unknown" />
            </ul>
          </div>
        </div>
      </div>
    </>
  );
}

function StatusRow({
  label,
  status,
}: {
  label: string;
  status: "ok" | "error" | "unknown";
}) {
  const color =
    status === "ok"
      ? "bg-green-500"
      : status === "error"
        ? "bg-red-500"
        : "bg-gray-300";
  return (
    <li className="flex items-center justify-between">
      <span className="text-gray-700">{label}</span>
      <span className={`w-2.5 h-2.5 rounded-full ${color}`} />
    </li>
  );
}
