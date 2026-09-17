"use client";

import Link from "next/link";
import { AuthShell } from "@/components/layout/auth-shell";

export default function ForgotPasswordPage() {
  return (
    <AuthShell
      title="Şifre kurtarma"
      subtitle="Hesabınıza yeniden erişmek için aşağıdaki adımları izleyin."
    >
      <div className="card space-y-4 text-sm leading-relaxed text-muted">
        <p>
          Miyuna&apos;da oturum açmak için kayıtlı e-posta adresiniz ve şifreniz gerekir. Şifrenizi hatırlamıyorsanız
          organizasyon yöneticinizden veya hesap sahibinden destek isteyin.
        </p>
        <p>
          Giriş sırasında e-posta doğrulaması veya MFA istenebilir; bu adımlar tamamlandığında oturumunuz açılır.
        </p>
        <p className="text-xs">
          Kalıcı şifre sıfırlama akışı yakında eklenecek. Acil erişim gerekiyorsa güvenlik ayarlarından oturum
          yönetimini kontrol edin.
        </p>
      </div>

      <div className="flex flex-wrap gap-3 text-sm">
        <Link href="/auth/login" className="btn-primary">
          Giriş ekranına dön
        </Link>
        <Link href="/settings/security" className="btn-secondary">
          Güvenlik ayarları
        </Link>
      </div>
    </AuthShell>
  );
}
