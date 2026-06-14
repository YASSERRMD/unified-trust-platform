import { TopBar } from "@/components/layout/TopBar";

export default function JITPage() {
  return (
    <>
      <TopBar title="JIT Access" subtitle="Just-in-time access request workflow" />
      <div className="p-6">
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center text-gray-500 text-sm">
          JIT access request UI coming in a future release.
          <br />
          Use the API at <code className="font-mono text-xs">/api/v1/jit/requests</code>.
        </div>
      </div>
    </>
  );
}
