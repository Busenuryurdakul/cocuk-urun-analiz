"use client";

import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
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
    <AppShell
      title="Uyumluluk"
      kicker="Organizasyon"
      orgId={orgId}
      description="Compliance Engine her zaman açıktır; OFF profili desteklenmez."
    >
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && (
        <AsyncView
          state="retry"
          retry={
            <button type="button" onClick={() => void load()} className="btn-secondary">
              Tekrar dene
            </button>
          }
        />
      )}
      {view === "success" && policy && (
        <div className="space-y-6">
          <section className="card text-sm">
            <p className="kicker">Aktif politika</p>
            <p className="mt-2 font-display text-2xl">
              {policy.profile} / {policy.version}
            </p>
            <p className="mt-2 text-muted">Durum: {policy.status}</p>
          </section>
          <form onSubmit={updateProfile} className="card max-w-lg space-y-3">
            <h2 className="font-display text-xl">Profil güncelle (OWNER/ADMIN)</h2>
            <select className="input" value={profile} onChange={(e) => setProfile(e.target.value)}>
              <option value="KVKK">KVKK</option>
              <option value="GDPR">GDPR</option>
              <option value="BOTH">BOTH</option>
            </select>
            <button type="submit" className="btn-primary">
              Güncelle
            </button>
          </form>
          <section className="card space-y-3">
            <h2 className="font-display text-xl">Onaylarım</h2>
            {consents.length === 0 ? (
              <AsyncView state="empty" />
            ) : (
              <ul className="space-y-2 text-sm">
                {consents.map((c) => (
                  <li key={c.id} className="flex items-center justify-between gap-3 rounded-xl bg-cream px-3 py-2">
                    <span>
                      {c.purpose} ({c.policyVersion})
                    </span>
                    {!c.withdrawnAt ? (
                      <button type="button" className="text-sm font-semibold text-red-700 underline" onClick={() => void withdrawConsent(c.purpose)}>
                        Geri çek
                      </button>
                    ) : (
                      <span className="text-muted">Geri çekildi</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
            <div className="flex flex-wrap gap-2 pt-2">
              {["REGISTRATION", "DATA_PROCESSING", "ORG_MEMBERSHIP"].map((p) => (
                <button key={p} type="button" className="btn-secondary !px-3 !py-1.5 text-xs" onClick={() => void grantConsent(p)}>
                  {p} onayla
                </button>
              ))}
            </div>
          </section>
        </div>
      )}
    </AppShell>
  );
}
