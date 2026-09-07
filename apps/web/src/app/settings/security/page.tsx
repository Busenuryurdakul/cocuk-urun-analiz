"use client";

import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { authErrorMessage, graphqlRequest } from "@/lib/graphql";

type Device = {
  id: string;
  platform: string;
  label: string;
  lastActiveAt: string;
  verified: boolean;
};

type Activity = {
  id: string;
  action: string;
  ipAddress?: string | null;
  timestamp: string;
};

type DeletionStep = "idle" | "code-sent" | "done";

export default function SecuritySettingsPage() {
  const [view, setView] = useState<"loading" | "error" | "success">("loading");
  const [email, setEmail] = useState("");
  const [devices, setDevices] = useState<Device[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [error, setError] = useState("");
  const [revoking, setRevoking] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [deletionStep, setDeletionStep] = useState<DeletionStep>("idle");
  const [deletionCode, setDeletionCode] = useState("");
  const [deletionBusy, setDeletionBusy] = useState(false);
  const [deletionMessage, setDeletionMessage] = useState("");

  useEffect(() => {
    graphqlRequest<{
      me: { email: string };
      myDevices: Device[];
      myActivityLog: Activity[];
    }>(`{
      me { email }
      myDevices { id platform label lastActiveAt verified }
      myActivityLog(limit: 30) { id action ipAddress timestamp }
    }`)
      .then((data) => {
        setEmail(data.me.email);
        setDevices(data.myDevices);
        setActivity(data.myActivityLog);
        setView("success");
      })
      .catch((err) => {
        setError(authErrorMessage(err, "Güvenlik bilgileri yüklenemedi"));
        setView("error");
      });
  }, []);

  async function revokeDevice(deviceId: string) {
    setRevoking(deviceId);
    try {
      await graphqlRequest(`mutation RevokeDevice($deviceId: ID!) {
        revokeDevice(deviceId: $deviceId)
      }`, { deviceId });
      setDevices((prev) => prev.filter((d) => d.id !== deviceId));
    } catch (err) {
      setError(authErrorMessage(err, "Cihaz iptal edilemedi"));
    } finally {
      setRevoking(null);
    }
  }

  async function downloadMyData() {
    setExporting(true);
    setError("");
    try {
      const data = await graphqlRequest<{ exportMyData: { schemaVersion: string; exportedAt: string; payload: string } }>(
        `mutation ExportMyData {
          exportMyData { schemaVersion exportedAt payload }
        }`
      );
      const blob = new Blob([data.exportMyData.payload], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `miyuna-data-export-${data.exportMyData.exportedAt.replace(/[:.]/g, "-")}.json`;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(authErrorMessage(err, "Veri dışa aktarılamadı"));
    } finally {
      setExporting(false);
    }
  }

  async function requestAccountDeletion() {
    setDeletionBusy(true);
    setDeletionMessage("");
    setError("");
    try {
      await graphqlRequest(`mutation RequestAccountDeletion {
        requestAccountDeletion { status expiresAt }
      }`);
      setDeletionStep("code-sent");
      setDeletionMessage("Onay kodu e-posta adresinize gönderildi. Kodu girerek silmeyi tamamlayın.");
    } catch (err) {
      setError(authErrorMessage(err, "Silme isteği gönderilemedi"));
    } finally {
      setDeletionBusy(false);
    }
  }

  async function confirmAccountDeletion() {
    if (!deletionCode.trim()) {
      setError("Onay kodu gerekli");
      return;
    }
    setDeletionBusy(true);
    setError("");
    try {
      await graphqlRequest(`mutation ConfirmAccountDeletion($code: String!) {
        confirmAccountDeletion(code: $code)
      }`, { code: deletionCode.trim() });
      setDeletionStep("done");
      setDeletionMessage("Hesabınız silindi. Oturumunuz kapatıldı.");
      window.location.href = "/auth/login";
    } catch (err) {
      const msg = authErrorMessage(err, "Hesap silinemedi");
      setError(msg);
      if (msg.toLowerCase().includes("ownership") || msg.includes("OWNERSHIP")) {
        setDeletionMessage("Takım organizasyonlarında sahip olduğunuz workspace varsa önce sahipliği devretmelisiniz.");
      }
    } finally {
      setDeletionBusy(false);
    }
  }

  if (view === "loading") {
    return (
      <AppShell title="Güvenlik" kicker="Hesap">
        <AsyncView state="loading" />
      </AppShell>
    );
  }

  if (view === "error") {
    return (
      <AppShell title="Güvenlik" kicker="Hesap">
        <AsyncView state="error" error={<p className="alert-error">{error}</p>} />
      </AppShell>
    );
  }

  return (
    <AppShell title="Güvenlik" kicker="Hesap" description="Aktif cihazlarınız, veri dışa aktarma ve hesap silme." accountEmail={email}>
      {error && <p className="alert-error mb-4">{error}</p>}
      <div className="grid gap-6 lg:grid-cols-2">
        <section className="card space-y-4">
          <h2 className="text-lg font-semibold text-forest">Aktif cihazlar</h2>
          {devices.length === 0 && <p className="text-sm text-muted">Kayıtlı cihaz yok.</p>}
          <ul className="space-y-3">
            {devices.map((device) => (
              <li key={device.id} className="rounded-lg border border-clay/30 p-3">
                <p className="font-medium text-forest">{device.label || device.platform}</p>
                <p className="text-xs text-muted">Son aktivite: {new Date(device.lastActiveAt).toLocaleString("tr-TR")}</p>
                <button
                  type="button"
                  disabled={revoking === device.id}
                  onClick={() => void revokeDevice(device.id)}
                  className="btn-danger mt-2 text-sm"
                >
                  {revoking === device.id ? "İptal ediliyor…" : "Bu cihazı iptal et"}
                </button>
              </li>
            ))}
          </ul>
        </section>

        <section className="card space-y-4">
          <h2 className="text-lg font-semibold text-forest">Aktivite geçmişi</h2>
          {activity.length === 0 && <p className="text-sm text-muted">Henüz aktivite kaydı yok.</p>}
          <ul className="max-h-96 space-y-2 overflow-y-auto text-sm">
            {activity.map((entry) => (
              <li key={entry.id} className="border-b border-clay/20 pb-2">
                <span className="font-medium text-forest">{entry.action}</span>
                {entry.ipAddress && <span className="text-muted"> — {entry.ipAddress}</span>}
                <p className="text-xs text-muted">{new Date(entry.timestamp).toLocaleString("tr-TR")}</p>
              </li>
            ))}
          </ul>
        </section>

        <section className="card space-y-4 lg:col-span-2">
          <h2 className="text-lg font-semibold text-forest">Veri ve hesap</h2>
          <p className="text-sm text-muted">KVKK/GDPR kapsamında kişisel verilerinizi indirebilir veya hesabınızı kalıcı olarak silebilirsiniz.</p>
          <div className="flex flex-wrap gap-3">
            <button type="button" disabled={exporting} onClick={() => void downloadMyData()} className="btn-secondary">
              {exporting ? "Hazırlanıyor…" : "Verilerimi indir"}
            </button>
          </div>
        </section>

        <section className="card space-y-4 lg:col-span-2 border-red-200">
          <h2 className="text-lg font-semibold text-red-800">Hesabı sil</h2>
          <p className="text-sm text-muted">
            Bu işlem geri alınamaz. Oturumlarınız ve cihazlarınız iptal edilir; kimlik bilgileriniz anonimleştirilir.
            Takım organizasyonlarında tek sahip olduğunuz workspace varsa önce sahipliği devretmeniz gerekir.
          </p>
          {deletionStep === "idle" && (
            <button type="button" disabled={deletionBusy} onClick={() => void requestAccountDeletion()} className="btn-danger">
              {deletionBusy ? "Gönderiliyor…" : "Silme onay kodu gönder"}
            </button>
          )}
          {deletionStep === "code-sent" && (
            <div className="space-y-3 max-w-sm">
              <label className="block text-sm font-medium text-forest" htmlFor="deletion-code">
                E-postanıza gelen 6 haneli kod
              </label>
              <input
                id="deletion-code"
                type="text"
                inputMode="numeric"
                maxLength={6}
                value={deletionCode}
                onChange={(e) => setDeletionCode(e.target.value)}
                className="input w-full"
                placeholder="000000"
              />
              <button type="button" disabled={deletionBusy} onClick={() => void confirmAccountDeletion()} className="btn-danger">
                {deletionBusy ? "Siliniyor…" : "Hesabımı kalıcı olarak sil"}
              </button>
            </div>
          )}
          {deletionMessage && <p className="text-sm text-forest">{deletionMessage}</p>}
        </section>
      </div>
    </AppShell>
  );
}
