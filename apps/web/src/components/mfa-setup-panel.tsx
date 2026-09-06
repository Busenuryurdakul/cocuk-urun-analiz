"use client";

import { FormEvent } from "react";

type MfaSetupPanelProps = {
  secret: string;
  otpauthUrl: string;
  submitLabel?: string;
  onSubmit: (code: string) => Promise<void>;
};

function qrCodeUrl(data: string): string {
  return `https://api.qrserver.com/v1/create-qr-code/?size=220x220&margin=10&data=${encodeURIComponent(data)}`;
}

export function MfaSetupPanel({ secret, otpauthUrl, submitLabel = "Doğrula", onSubmit }: MfaSetupPanelProps) {
  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const code = String(new FormData(e.currentTarget).get("code") ?? "");
    await onSubmit(code);
  }

  return (
    <div className="card space-y-4">
      <p className="text-sm text-muted">
        Google Authenticator, Authy veya benzeri bir uygulama ile aşağıdaki QR kodu okutun.
      </p>
      <div className="flex justify-center">
        <img
          src={qrCodeUrl(otpauthUrl)}
          alt="MFA QR kodu"
          width={220}
          height={220}
          className="rounded-xl border border-cream bg-white p-2"
        />
      </div>
      <p className="text-xs text-muted">
        QR okuyamıyorsanız secret&apos;ı elle girin:
      </p>
      <code className="block break-all rounded-xl bg-cream p-3 text-xs">{secret}</code>
      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-3">
        <label className="label">
          Authenticator kodu
          <input name="code" required className="input" autoComplete="one-time-code" inputMode="numeric" />
        </label>
        <button type="submit" className="btn-primary w-full">
          {submitLabel}
        </button>
      </form>
    </div>
  );
}
