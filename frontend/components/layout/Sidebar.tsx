"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { href: "/admin", label: "Dashboard", icon: "⊞" },
  { href: "/admin/tenants", label: "Tenants", icon: "🏢" },
  { href: "/admin/users", label: "Users", icon: "👥" },
  { href: "/admin/roles", label: "Roles & Permissions", icon: "🔑" },
  { href: "/admin/policies", label: "Policies", icon: "📋" },
  { href: "/admin/delegations", label: "Delegations", icon: "🤝" },
  { href: "/admin/jit", label: "JIT Access", icon: "⏱" },
  { href: "/admin/federation", label: "Federation", icon: "🔗" },
  { href: "/admin/audit", label: "Audit Log", icon: "📜" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-gray-900 text-white flex flex-col h-screen sticky top-0">
      <div className="p-4 border-b border-gray-700">
        <h1 className="text-lg font-bold tracking-tight">Unified Trust</h1>
        <p className="text-xs text-gray-400 mt-0.5">Admin Portal</p>
      </div>
      <nav className="flex-1 py-4 overflow-y-auto">
        <ul className="space-y-0.5 px-2">
          {navItems.map((item) => {
            const active =
              item.href === "/admin"
                ? pathname === "/admin"
                : pathname.startsWith(item.href);
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                    active
                      ? "bg-blue-600 text-white"
                      : "text-gray-300 hover:bg-gray-800 hover:text-white"
                  }`}
                >
                  <span className="text-base">{item.icon}</span>
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
      <div className="p-4 border-t border-gray-700 text-xs text-gray-500">
        v1.0.0
      </div>
    </aside>
  );
}
