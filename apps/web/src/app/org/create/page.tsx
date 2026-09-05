"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";

type Organization = {
  id: string;
  name: string;
  complianceProfile: string;
};

export default function CreateOrganizationPage() {
  const [name, setName] = useState("");
  const [profile, setProfile] = useState("KVKK");
  const [state, setState] = useState<"idle" | "loading" | "success" | "error" | "unauthorized">("idle");
  const [org, setOrg] = useState<Organization | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setState("loading");
    try {
      const data = await graphqlRequest<{ createOrganization: Organization }>(
        `mutation($input: CreateOrganizationInput!) {
          createOrganization(input: $input) { id name complianceProfile }
        }`,
        { input: { name, complianceProfile: profile } },
      );
      setOrg(data.createOrganization);
      setState("success");
    } catch (err) {
      if (err instanceof Error && err.message === "UNAUTHORIZED") {
        setState("unauthorized");
      } else {
        setState("error");
      }
    }
  }

  return (
    <AppShell
      title="Organizasyon oluştur"
      kicker="Çalışma alanı"
      description="Uyumluluk profili seçilir; Compliance Engine kapatılamaz."
    >
      {state === "loading" && <AsyncView state="loading" />}
      {state === "unauthorized" && (
        <AsyncView
          state="unauthorized"
          unauthorized={<Link href="/auth/login" className="btn-primary">Giriş yap</Link>}
        />
      )}
      {state === "error" && <AsyncView state="error" />}
      {state === "success" && org && (
        <div className="alert-success">
          <p>Oluşturuldu: {org.name}</p>
          <Link href={`/org/${org.id}/members`} className="mt-3 inline-block font-semibold underline">
            Üyeleri yönet
          </Link>
        </div>
      )}
      {(state === "idle" || state === "error") && (
        <form onSubmit={onSubmit} className="card max-w-lg space-y-4">
          <label className="label">
            Ad
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
          </label>
          <label className="label">
            Uyumluluk profili
            <select className="input" value={profile} onChange={(e) => setProfile(e.target.value)}>
              <option value="KVKK">KVKK</option>
              <option value="GDPR">GDPR</option>
              <option value="BOTH">BOTH</option>
            </select>
          </label>
          <button type="submit" className="btn-primary">
            Oluştur
          </button>
        </form>
      )}
    </AppShell>
  );
}
