"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { graphqlRequest } from "@/lib/graphql";

type Policy = {
  profile: string;
  version: string;
  status: string;
};

type Consent = {
  id: string;
  purpose: string;
  policyVersion: string;
  grantedAt: string;
  withdrawnAt?: string | null;
};

export default function OrgCompliancePage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const [policy, setPolicy] = useState<Policy | null>(null);
  const [consents, setConsents] = useState<Consent[]>([]);
  const [profile, setProfile] = useState("KVKK");
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized">("loading");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const data = await graphqlRequest<{
        activeCompliancePolicy: Policy;
        myConsents: Consent[];
        organization: { complianceProfile: string; compliancePolicyVersion: string };
      }>(`query($id: ID!) {
        organization(organizationId: $id) { complianceProfile compliancePolicyVersion }
        activeCompliancePolicy(organizationId: $id) { profile version status }
        myConsents { id purpose policyVersion grantedAt withdrawnAt }
      }`, { id: orgId });
      setPolicy(data.activeCompliancePolicy);
      setConsents(data.myConsents);
      setProfile(data.organization.complianceProfile);
      setView("success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else {
        setView("error");
      }
    }
  }, [orgId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function updateProfile(e: FormEvent) {
    e.preventDefault();
    await graphqlRequest(
      `mutation($input: UpdateComplianceProfileInput!) {
        updateOrganizationComplianceProfile(input: $input) { id complianceProfile }
      }`,
      { input: { organizationId: orgId, complianceProfile: profile } },
    );
    await load();
  }

  async function grantConsent(purpose: string) {
    await graphqlRequest(
      `mutation($input: GrantConsentInput!) { grantConsent(input: $input) { id purpose } }`,
      { input: { purpose, organizationId: orgId } },
    );
    await load();
  }

  async function withdrawConsent(purpose: string) {
    await graphqlRequest(
      `mutation($input: WithdrawConsentInput!) { withdrawConsent(input: $input) }`,
      { input: { purpose, organizationId: orgId } },
    );
    await load();
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Uyumluluk</h1>
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && <AsyncView state="retry" retry={<button type="button" onClick={() => void load()} className="underline">Tekrar dene</button>} />}
      {view === "success" && policy && (
        <>
          <section className="rounded-xl border bg-white p-4 text-sm">
            <p>Aktif politika: {policy.profile} / {policy.version}</p>
            <p className="text-slate-500">Compliance Engine her zaman açıktır; OFF profili desteklenmez.</p>
          </section>
          <form onSubmit={updateProfile} className="space-y-3 rounded-xl border bg-white p-4">
            <h2 className="font-medium">Profil güncelle (OWNER/ADMIN)</h2>
            <select className="w-full rounded border px-3 py-2" value={profile} onChange={(e) => setProfile(e.target.value)}>
              <option value="KVKK">KVKK</option>
              <option value="GDPR">GDPR</option>
              <option value="BOTH">BOTH</option>
            </select>
            <button type="submit" className="rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">Güncelle</button>
          </form>
          <section className="space-y-2 rounded-xl border bg-white p-4">
            <h2 className="font-medium">Onaylarım</h2>
            {consents.length === 0 ? <AsyncView state="empty" /> : (
              <ul className="space-y-2 text-sm">
                {consents.map((c) => (
                  <li key={c.id} className="flex items-center justify-between">
                    <span>{c.purpose} ({c.policyVersion})</span>
                    {!c.withdrawnAt ? (
                      <button type="button" className="text-red-600 underline" onClick={() => void withdrawConsent(c.purpose)}>Geri çek</button>
                    ) : (
                      <span className="text-slate-400">Geri çekildi</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
            <div className="flex flex-wrap gap-2 pt-2">
              {["REGISTRATION", "DATA_PROCESSING", "ORG_MEMBERSHIP"].map((p) => (
                <button key={p} type="button" className="rounded border px-2 py-1 text-xs" onClick={() => void grantConsent(p)}>
                  {p} onayla
                </button>
              ))}
            </div>
          </section>
        </>
      )}
      <Link href={`/org/${orgId}/members`} className="text-sm text-slate-500 underline">Üyeler</Link>
    </main>
  );
}
