import { TopBar } from "@/components/layout/TopBar";

export default function FederationPage() {
  return (
    <>
      <TopBar title="Federation" subtitle="External OIDC identity provider configuration" />
      <div className="p-6">
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center text-gray-500 text-sm">
          Federation provider management UI coming in a future release.
          <br />
          Use the API at{" "}
          <code className="font-mono text-xs">/api/v1/federation/providers</code>.
        </div>
      </div>
    </>
  );
}
