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

export default function SecuritySettingsPage() {
  const [view, setView] = useState<"loading" | "error" | "success">("loading");
  const [email, setEmail] = useState("");
  const [devices, setDevices] = useState<Device[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [error, setError] = useState("");
  const [revoking, setRevoking] = useState<string | null>(null);

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
    <AppShell title="Güvenlik" kicker="Hesap" description="Aktif cihazlarınız ve oturum aktiviteleriniz." accountEmail={email}>
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
      </div>
    </AppShell>
  );
}
