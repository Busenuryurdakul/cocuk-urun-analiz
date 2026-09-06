"use client";

import { FormEvent, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams } from "next/navigation";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { UsagePanel, type UsageDashboardData } from "@/components/llm/usage-panel";
import { graphqlRequest } from "@/lib/graphql";
import { monthStartIsoDate } from "@/lib/llm-events";

type Model = {
  modelKey: string;
  displayName: string;
  status: string;
  healthStatus: string;
  defaultForPlatform: boolean;
  fallbackForPlatform: boolean;
};

type Health = {
  modelKey: string;
  displayName: string;
  healthStatus: string;
  providerKey: string;
};

type Config = {
  id: string;
  routingPolicyVersion: string;
  personaKey: string;
  defaultModelKey: string;
  fallbackModelKey: string;
  publishedAt: string;
  reason?: string;
};

type Draft = {
  id: string;
  status: string;
  routingPolicyVersion: string;
  personaKey: string;
  personaVersion: string;
  defaultModelKey: string;
  fallbackModelKey: string;
  validationErrors: string[];
  updatedAt: string;
};

type OrgSettings = {
  organizationId: string;
  personaKey: string;
  defaultModelKey: string;
  fallbackModelKey: string;
};

export default function OrgLLMPage() {
  return (
    <Suspense fallback={<AsyncView state="loading" />}>
      <OrgLLMPageContent />
    </Suspense>
  );
}

function OrgLLMPageContent() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const orgId = params.id;
  const fromDate = useMemo(() => monthStartIsoDate(), []);
  const highlightedCallId = searchParams.get("call");

  const [models, setModels] = useState<Model[]>([]);
  const [health, setHealth] = useState<Health[]>([]);
  const [config, setConfig] = useState<Config | null>(null);
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [snapshotHistory, setSnapshotHistory] = useState<Config[]>([]);
  const [draftReason, setDraftReason] = useState("Organizasyon LLM yapılandırması güncellemesi");
  const [rollbackSnapshotId, setRollbackSnapshotId] = useState("");
  const [rollbackReason, setRollbackReason] = useState("Önceki yapılandırmaya geri dön");
  const [configBusy, setConfigBusy] = useState(false);
  const [orgSettings, setOrgSettings] = useState<OrgSettings | null>(null);
  const [usage, setUsage] = useState<UsageDashboardData | null>(null);
  const [personaKey, setPersonaKey] = useState("careful_analyst");
  const [defaultModelKey, setDefaultModelKey] = useState("careful_analyst");
  const [fallbackModelKey, setFallbackModelKey] = useState("result_analyst");
  const [testPrompt, setTestPrompt] = useState("Ürün analizi için kısa bir özet üret.");
  const [testResult, setTestResult] = useState("");
  const [saveMessage, setSaveMessage] = useState("");
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized">("loading");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const data = await graphqlRequest<{
        llmModels: Model[];
        llmHealth: Health[];
        activeLLMConfiguration: Config | null;
        llmConfigurationDrafts: Draft[];
        llmOrgSettings: OrgSettings | null;
        llmUsageDashboard: UsageDashboardData;
      }>(
        `query($id: ID!, $fromDate: String!) {
          llmModels(organizationId: $id) { modelKey displayName status healthStatus defaultForPlatform fallbackForPlatform }
          llmHealth(organizationId: $id) { modelKey displayName healthStatus providerKey }
          activeLLMConfiguration(organizationId: $id) {
            id routingPolicyVersion personaKey defaultModelKey fallbackModelKey publishedAt reason
          }
          llmConfigurationDrafts(organizationId: $id, limit: 10) {
            id status routingPolicyVersion personaKey personaVersion defaultModelKey fallbackModelKey validationErrors updatedAt
          }
          llmOrgSettings(organizationId: $id) { organizationId personaKey defaultModelKey fallbackModelKey }
          llmUsageDashboard(organizationId: $id, fromDate: $fromDate, recentLimit: 12) {
            summary { callCount inputTokens outputTokens totalTokens estimatedCostUsd fallbackCount }
            byModel { modelKey displayName callCount inputTokens outputTokens totalTokens estimatedCostUsd }
            recentCalls { id modelKey personaKey routingReason fallbackUsed inputTokens outputTokens latencyMs status createdAt }
          }
        }`,
        { id: orgId, fromDate },
      );
      setModels(data.llmModels);
      setHealth(data.llmHealth);
      setConfig(data.activeLLMConfiguration);
      setDrafts(data.llmConfigurationDrafts);
      if (data.activeLLMConfiguration) {
        const historyKey = `miyuna:llm-snapshots:${orgId}`;
        const stored = typeof window !== "undefined" ? window.localStorage.getItem(historyKey) : null;
        const previous = stored ? (JSON.parse(stored) as Config[]) : [];
        const merged = [
          data.activeLLMConfiguration,
          ...previous.filter((item) => item.id !== data.activeLLMConfiguration?.id),
        ].slice(0, 8);
        setSnapshotHistory(merged);
        if (typeof window !== "undefined") {
          window.localStorage.setItem(historyKey, JSON.stringify(merged));
        }
      }
      setOrgSettings(data.llmOrgSettings);
      setUsage(data.llmUsageDashboard);
      const settings = data.llmOrgSettings;
      if (settings?.personaKey) setPersonaKey(settings.personaKey);
      if (settings?.defaultModelKey) setDefaultModelKey(settings.defaultModelKey);
      if (settings?.fallbackModelKey) setFallbackModelKey(settings.fallbackModelKey);
      setView("success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else {
        setView("error");
      }
    }
  }, [fromDate, orgId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function createDraft() {
    setConfigBusy(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await graphqlRequest(
        `mutation($input: CreateLLMConfigurationDraftInput!) {
          createLLMConfigurationDraft(input: $input) { id status }
        }`,
        {
          input: {
            organizationId: orgId,
            personaKey,
            defaultModelKey,
            fallbackModelKey,
            reason: draftReason,
          },
        },
      );
      setSaveMessage("Taslak oluşturuldu.");
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Taslak oluşturulamadı");
    } finally {
      setConfigBusy(false);
    }
  }

  async function validateDraft(draftId: string) {
    setConfigBusy(true);
    setSaveError("");
    setSaveMessage("");
    try {
      const data = await graphqlRequest<{ validateLLMConfiguration: Draft }>(
        `mutation($input: ValidateLLMConfigurationInput!) {
          validateLLMConfiguration(input: $input) { id status validationErrors }
        }`,
        { input: { organizationId: orgId, draftId } },
      );
      if (data.validateLLMConfiguration.validationErrors.length > 0) {
        setSaveError(data.validateLLMConfiguration.validationErrors.join(", "));
      } else {
        setSaveMessage("Taslak doğrulandı.");
      }
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Doğrulama başarısız");
    } finally {
      setConfigBusy(false);
    }
  }

  async function publishDraft(draftId: string) {
    setConfigBusy(true);
    setSaveError("");
    setSaveMessage("");
    try {
      const data = await graphqlRequest<{ publishLLMConfiguration: Config }>(
        `mutation($input: PublishLLMConfigurationInput!) {
          publishLLMConfiguration(input: $input) {
            id routingPolicyVersion personaKey defaultModelKey fallbackModelKey publishedAt reason
          }
        }`,
        { input: { organizationId: orgId, draftId, reason: draftReason } },
      );
      const published = data.publishLLMConfiguration;
      const historyKey = `miyuna:llm-snapshots:${orgId}`;
      const merged = [published, ...snapshotHistory.filter((item) => item.id !== published.id)].slice(0, 8);
      setSnapshotHistory(merged);
      if (typeof window !== "undefined") {
        window.localStorage.setItem(historyKey, JSON.stringify(merged));
      }
      setSaveMessage(`Yapılandırma yayınlandı (snapshot ${published.id}).`);
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Yayınlama başarısız");
    } finally {
      setConfigBusy(false);
    }
  }

  async function rollbackSnapshot(e: FormEvent) {
    e.preventDefault();
    if (!rollbackSnapshotId.trim()) return;
    setConfigBusy(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await graphqlRequest(
        `mutation($input: RollbackLLMConfigurationInput!) {
          rollbackLLMConfiguration(input: $input) { id publishedAt }
        }`,
        {
          input: {
            organizationId: orgId,
            snapshotId: rollbackSnapshotId.trim(),
            reason: rollbackReason,
          },
        },
      );
      setSaveMessage("Yapılandırma geri alındı.");
      setRollbackSnapshotId("");
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Geri alma başarısız");
    } finally {
      setConfigBusy(false);
    }
  }

  async function savePersona(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await graphqlRequest(
        `mutation($input: SetOrganizationPersonaInput!) {
          setOrganizationPersona(input: $input) { organizationId personaKey }
        }`,
        { input: { organizationId: orgId, personaKey } },
      );
      setSaveMessage("Persona kaydedildi.");
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Persona kaydedilemedi");
    } finally {
      setSaving(false);
    }
  }

  async function saveModels(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await graphqlRequest(
        `mutation($input: SetOrganizationModelsInput!) {
          setOrganizationModels(input: $input) { organizationId defaultModelKey fallbackModelKey }
        }`,
        { input: { organizationId: orgId, defaultModelKey, fallbackModelKey } },
      );
      setSaveMessage("Model tercihleri kaydedildi.");
      await load();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Model tercihleri kaydedilemedi");
    } finally {
      setSaving(false);
    }
  }

  async function runTest(e: FormEvent) {
    e.preventDefault();
    setTesting(true);
    setTestResult("");
    setSaveError("");
    try {
      const data = await graphqlRequest<{
        testLLMConfiguration: {
          contentPreview: string;
          modelKey: string;
          routingReason: string;
          fallbackUsed: boolean;
          escalationUsed: boolean;
          inputTokens: number;
          outputTokens: number;
        };
      }>(
        `mutation($input: TestLLMConfigurationInput!) {
          testLLMConfiguration(input: $input) {
            contentPreview modelKey routingReason fallbackUsed escalationUsed inputTokens outputTokens
          }
        }`,
        { input: { organizationId: orgId, personaKey, prompt: testPrompt } },
      );
      const result = data.testLLMConfiguration;
      setTestResult(
        [
          `Model: ${result.modelKey}`,
          `Fallback: ${result.fallbackUsed ? "evet" : "hayır"}`,
          `Yükseltme: ${result.escalationUsed ? "evet" : "hayır"}`,
          `Token: ${result.inputTokens + result.outputTokens}`,
          result.routingReason,
          "",
          result.contentPreview,
        ].join("\n"),
      );
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : "Test başarısız");
    } finally {
      setTesting(false);
    }
  }

  const modelName = (key: string) => models.find((model) => model.modelKey === key)?.displayName ?? key;

  const draftPreview = {
    personaKey,
    defaultModelKey,
    fallbackModelKey,
  };

  function compareRow(label: string, oldValue: string, newValue: string) {
    const changed = oldValue !== newValue;
    return (
      <li className="grid gap-2 rounded-xl border border-sand/70 px-3 py-2 text-sm sm:grid-cols-3">
        <span className="text-muted">{label}</span>
        <span className={changed ? "line-through text-muted" : ""}>{oldValue || "—"}</span>
        <span className={changed ? "font-medium text-forest" : ""}>{newValue || "—"}</span>
      </li>
    );
  }

  return (
    <AppShell
      orgId={orgId}
      title="LLM kontrol paneli"
      kicker="Yapay zeka"
      description="Hızlı model ile başlayın, uzun süreçlerde ağır modele yükseltin. Kullanımı Cursor benzeri panelden izleyin."
    >
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && (
        <AsyncView
          state="retry"
          retry={
            <button type="button" className="btn-secondary" onClick={() => void load()}>
              Tekrar dene
            </button>
          }
        />
      )}

      {view === "success" && (
        <div className="space-y-6">
          {usage && <UsagePanel orgId={orgId} usage={usage} />}

          <section className="card grid gap-5 lg:grid-cols-2">
            <div>
              <p className="kicker">Yayınlanan yapılandırma</p>
              {config ? (
                <ul className="mt-3 space-y-2 text-sm">
                  <li className="flex justify-between gap-3">
                    <span className="text-muted">Routing</span>
                    <span className="font-medium">{config.routingPolicyVersion}</span>
                  </li>
                  <li className="flex justify-between gap-3">
                    <span className="text-muted">Persona</span>
                    <span className="font-medium">{config.personaKey}</span>
                  </li>
                  <li className="flex justify-between gap-3">
                    <span className="text-muted">Varsayılan model</span>
                    <span className="font-medium">{modelName(config.defaultModelKey)}</span>
                  </li>
                  <li className="flex justify-between gap-3">
                    <span className="text-muted">Yedek / ağır model</span>
                    <span className="font-medium">{modelName(config.fallbackModelKey)}</span>
                  </li>
                </ul>
              ) : (
                <p className="mt-3 text-sm text-muted">Henüz yayınlanmış LLM yapılandırması yok.</p>
              )}
            </div>

            <div>
              <p className="kicker">Sağlık durumu</p>
              <ul className="mt-3 space-y-2">
                {health.map((item) => (
                  <li
                    key={item.modelKey}
                    className="flex items-center justify-between rounded-2xl border border-sand bg-cream/50 px-4 py-3 text-sm"
                  >
                    <div>
                      <p className="font-medium">{item.displayName}</p>
                      <p className="text-xs text-muted">{item.providerKey}</p>
                    </div>
                    <span className={item.healthStatus === "HEALTHY" ? "badge-forest" : "badge-muted"}>
                      {item.healthStatus}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </section>

          <section className="card">
            <p className="kicker">Kayıtlı modeller</p>
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b border-sand text-left text-xs uppercase tracking-wide text-muted">
                    <th className="py-2 pr-4">Anahtar</th>
                    <th className="py-2 pr-4">Ad</th>
                    <th className="py-2 pr-4">Durum</th>
                    <th className="py-2 pr-4">Sağlık</th>
                    <th className="py-2 pr-4">Rol</th>
                  </tr>
                </thead>
                <tbody>
                  {models.map((model) => (
                    <tr key={model.modelKey} className="border-b border-sand/70">
                      <td className="py-3 pr-4 font-mono text-xs">{model.modelKey}</td>
                      <td className="py-3 pr-4">{model.displayName}</td>
                      <td className="py-3 pr-4">{model.status}</td>
                      <td className="py-3 pr-4">{model.healthStatus}</td>
                      <td className="py-3 pr-4">
                        {model.defaultForPlatform && <span className="badge-forest mr-1">Hızlı</span>}
                        {model.fallbackForPlatform && <span className="badge-clay">Ağır</span>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section className="card space-y-5">
            <div>
              <p className="kicker">Manual control</p>
              <h2 className="font-display text-xl">Yapılandırma yönetimi</h2>
              <p className="mt-1 text-sm text-muted">
                Taslak oluştur → doğrula → yayınla. Otomatik job routing policy üzerinden çalışır; bu panel snapshot
                sürümlemesini yönetir.
              </p>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <div className="rounded-2xl border border-sand bg-cream/40 p-4">
                <p className="text-xs uppercase tracking-wide text-muted">OLD (aktif snapshot)</p>
                {config ? (
                  <ul className="mt-3 space-y-1 text-sm">
                    <li>Persona: {config.personaKey}</li>
                    <li>Model: {modelName(config.defaultModelKey)}</li>
                    <li>Yedek: {modelName(config.fallbackModelKey)}</li>
                    <li className="font-mono text-xs text-muted">{config.id}</li>
                  </ul>
                ) : (
                  <p className="mt-3 text-sm text-muted">Aktif snapshot yok.</p>
                )}
              </div>
              <div className="rounded-2xl border border-forest/30 bg-white p-4">
                <p className="text-xs uppercase tracking-wide text-muted">NEW (taslak önizleme)</p>
                <ul className="mt-3 space-y-1 text-sm">
                  <li>Persona: {draftPreview.personaKey}</li>
                  <li>Model: {modelName(draftPreview.defaultModelKey)}</li>
                  <li>Yedek: {modelName(draftPreview.fallbackModelKey)}</li>
                </ul>
              </div>
            </div>

            {config && (
              <ul className="space-y-2">
                {compareRow("Persona", config.personaKey, draftPreview.personaKey)}
                {compareRow("Varsayılan model", modelName(config.defaultModelKey), modelName(draftPreview.defaultModelKey))}
                {compareRow("Yedek model", modelName(config.fallbackModelKey), modelName(draftPreview.fallbackModelKey))}
              </ul>
            )}

            <label className="label">
              Değişiklik gerekçesi
              <input className="input" value={draftReason} onChange={(e) => setDraftReason(e.target.value)} />
            </label>

            <div className="flex flex-wrap gap-2">
              <button type="button" className="btn-primary" disabled={configBusy} onClick={() => void createDraft()}>
                Taslak oluştur
              </button>
            </div>

            {drafts.length > 0 && (
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b border-sand text-left text-xs uppercase tracking-wide text-muted">
                      <th className="py-2 pr-4">Durum</th>
                      <th className="py-2 pr-4">Persona</th>
                      <th className="py-2 pr-4">Modeller</th>
                      <th className="py-2 pr-4">İşlem</th>
                    </tr>
                  </thead>
                  <tbody>
                    {drafts.map((draft) => (
                      <tr key={draft.id} className="border-b border-sand/70">
                        <td className="py-3 pr-4">{draft.status}</td>
                        <td className="py-3 pr-4">{draft.personaKey}</td>
                        <td className="py-3 pr-4">
                          {modelName(draft.defaultModelKey)} → {modelName(draft.fallbackModelKey)}
                        </td>
                        <td className="py-3 pr-4">
                          <div className="flex flex-wrap gap-2">
                            {draft.status === "DRAFT" && (
                              <button
                                type="button"
                                className="btn-ghost !px-2 !py-1 text-xs"
                                disabled={configBusy}
                                onClick={() => void validateDraft(draft.id)}
                              >
                                Doğrula
                              </button>
                            )}
                            {draft.status === "VALIDATED" && (
                              <button
                                type="button"
                                className="btn-secondary !px-2 !py-1 text-xs"
                                disabled={configBusy}
                                onClick={() => void publishDraft(draft.id)}
                              >
                                Yayınla
                              </button>
                            )}
                          </div>
                          {draft.validationErrors.length > 0 && (
                            <p className="mt-1 text-xs text-red-700">{draft.validationErrors.join(", ")}</p>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <form onSubmit={(e) => void rollbackSnapshot(e)} className="grid gap-3 border-t border-sand pt-4 lg:grid-cols-3">
              <label className="label lg:col-span-1">
                Geri alınacak snapshot
                <select className="input" value={rollbackSnapshotId} onChange={(e) => setRollbackSnapshotId(e.target.value)}>
                  <option value="">Seçin…</option>
                  {snapshotHistory
                    .filter((item) => item.id !== config?.id)
                    .map((item) => (
                      <option key={item.id} value={item.id}>
                        {item.publishedAt.slice(0, 16)} — {item.personaKey}
                      </option>
                    ))}
                </select>
              </label>
              <label className="label lg:col-span-1">
                Geri alma gerekçesi
                <input className="input" value={rollbackReason} onChange={(e) => setRollbackReason(e.target.value)} />
              </label>
              <div className="flex items-end">
                <button type="submit" className="btn-secondary" disabled={configBusy || !rollbackSnapshotId}>
                  Snapshot&apos;a geri dön
                </button>
              </div>
            </form>
          </section>

          <section className="grid gap-6 lg:grid-cols-3">
            <form onSubmit={(e) => void saveModels(e)} className="card space-y-4">
              <div>
                <p className="kicker">Organizasyon</p>
                <h2 className="font-display text-xl">Manuel model tercihi</h2>
                <p className="mt-1 text-sm text-muted">
                  Snapshot yayınlamadan önce taslak alanlarını doldurur. Otomatik job routing&apos;i etkilemez.
                </p>
              </div>

              <label className="label">
                Başlangıç (hızlı)
                <select className="input" value={defaultModelKey} onChange={(e) => setDefaultModelKey(e.target.value)}>
                  {models.map((model) => (
                    <option key={model.modelKey} value={model.modelKey}>
                      {model.displayName}
                    </option>
                  ))}
                </select>
              </label>

              <label className="label">
                Yükseltme / inceleme (ağır)
                <select className="input" value={fallbackModelKey} onChange={(e) => setFallbackModelKey(e.target.value)}>
                  {models.map((model) => (
                    <option key={model.modelKey} value={model.modelKey}>
                      {model.displayName}
                    </option>
                  ))}
                </select>
              </label>

              <button type="submit" className="btn-primary" disabled={saving}>
                {saving ? "Kaydediliyor…" : "Modelleri kaydet"}
              </button>
            </form>

            <form onSubmit={(e) => void savePersona(e)} className="card space-y-4">
              <div>
                <p className="kicker">Organizasyon</p>
                <h2 className="font-display text-xl">Persona</h2>
                <p className="mt-1 text-sm text-muted">Agent davranış profili — analiz tonu ve talimatları.</p>
              </div>

              <label className="label">
                Persona
                <select className="input" value={personaKey} onChange={(e) => setPersonaKey(e.target.value)}>
                  <option value="careful_analyst">Careful Analyst</option>
                  <option value="result_analyst">Result Analyst</option>
                </select>
              </label>

              <button type="submit" className="btn-secondary" disabled={saving}>
                Persona kaydet
              </button>
            </form>

            <form onSubmit={(e) => void runTest(e)} className="card space-y-4">
              <div>
                <p className="kicker">Sandbox</p>
                <h2 className="font-display text-xl">Test konsolu</h2>
                <p className="mt-1 text-sm text-muted">Gateway üzerinden gerçek model çağrısı yapın.</p>
              </div>

              <label className="label">
                Prompt
                <textarea className="input min-h-28" value={testPrompt} onChange={(e) => setTestPrompt(e.target.value)} />
              </label>

              <button type="submit" className="btn-accent" disabled={testing}>
                {testing ? "Çalışıyor…" : "Test et"}
              </button>

              {testResult && (
                <pre className="whitespace-pre-wrap rounded-2xl bg-cream p-4 text-xs leading-relaxed">{testResult}</pre>
              )}
            </form>
          </section>

          {saveMessage && <p className="alert-success">{saveMessage}</p>}
          {saveError && <p className="alert-error">{saveError}</p>}

          {highlightedCallId && usage && (
            <section className="card">
              <p className="kicker">Çağrı vurgusu</p>
              <p className="mt-2 text-sm text-muted">
                Seçilen çağrı: <span className="font-mono text-ink">{highlightedCallId}</span>
              </p>
              {usage.recentCalls.some((call) => call.id === highlightedCallId) ? (
                <p className="mt-2 text-sm">Son çağrılar listesinde gösteriliyor.</p>
              ) : (
                <p className="mt-2 text-sm text-muted">Bu çağrı son 12 kayıtta yok — daha eski olabilir.</p>
              )}
            </section>
          )}

          {orgSettings && (
            <p className="text-center text-xs text-muted">
              Aktif org tercihi: {modelName(orgSettings.defaultModelKey)} → {modelName(orgSettings.fallbackModelKey)}
            </p>
          )}
        </div>
      )}
    </AppShell>
  );
}
