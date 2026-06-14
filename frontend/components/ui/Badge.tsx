interface BadgeProps {
  label: string;
  variant?: "green" | "red" | "yellow" | "blue" | "gray";
}

const variants: Record<string, string> = {
  green: "bg-green-100 text-green-800",
  red: "bg-red-100 text-red-800",
  yellow: "bg-yellow-100 text-yellow-800",
  blue: "bg-blue-100 text-blue-800",
  gray: "bg-gray-100 text-gray-800",
};

export function Badge({ label, variant = "gray" }: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${variants[variant]}`}
    >
      {label}
    </span>
  );
}

export function statusBadge(status: string) {
  const map: Record<string, BadgeProps["variant"]> = {
    active: "green",
    suspended: "red",
    pending: "yellow",
    inactive: "gray",
    permit: "green",
    deny: "red",
    success: "green",
    failure: "red",
    denied: "red",
  };
  return <Badge label={status} variant={map[status] ?? "gray"} />;
}
