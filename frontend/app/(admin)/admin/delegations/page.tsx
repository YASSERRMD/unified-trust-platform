import { TopBar } from "@/components/layout/TopBar";

export default function DelegationsPage() {
  return (
    <>
      <TopBar title="Delegations" subtitle="Time-bound role delegation between users" />
      <div className="p-6">
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center text-gray-500 text-sm">
          Delegation management UI coming in a future release.
          <br />
          Use the API at <code className="font-mono text-xs">/api/v1/delegations</code>.
        </div>
      </div>
    </>
  );
}
