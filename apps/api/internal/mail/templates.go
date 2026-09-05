package mail

import "fmt"

func VerificationEmail(verifyURL string) (subject, plain, html string) {
	subject = "Miyuna — e-posta doğrulama"
	plain = fmt.Sprintf("E-posta adresinizi doğrulamak için bağlantıyı açın:\n\n%s\n\nBağlantı 24 saat geçerlidir.", verifyURL)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head><meta charset="UTF-8"><title>%s</title></head>
<body style="font-family:system-ui,sans-serif;background:#f5f0e8;margin:0;padding:32px">
  <div style="max-width:480px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;border:1px solid #e0d8cc">
    <h1 style="color:#14352f;font-size:22px;margin:0 0 16px">Miyuna</h1>
    <p style="color:#333;line-height:1.6">Hesabınızı etkinleştirmek için e-posta adresinizi doğrulayın.</p>
    <p style="margin:24px 0"><a href="%s" style="display:inline-block;background:#14352f;color:#fff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:600">E-postamı doğrula</a></p>
    <p style="color:#666;font-size:13px">Bağlantı 24 saat geçerlidir. Bu isteği siz yapmadıysanız bu e-postayı yok sayın.</p>
  </div>
</body>
</html>`, subject, verifyURL)
	return subject, plain, html
}

func LoginOTPEmail(code string) (subject, plain, html string) {
	subject = "Miyuna — giriş doğrulama kodu"
	plain = fmt.Sprintf("Giriş doğrulama kodunuz: %s\n\nKod 10 dakika geçerlidir.", code)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head><meta charset="UTF-8"><title>%s</title></head>
<body style="font-family:system-ui,sans-serif;background:#f5f0e8;margin:0;padding:32px">
  <div style="max-width:480px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;border:1px solid #e0d8cc">
    <h1 style="color:#14352f;font-size:22px;margin:0 0 16px">Miyuna</h1>
    <p style="color:#333;line-height:1.6">Giriş yapmak için aşağıdaki 6 haneli kodu kullanın:</p>
    <p style="font-size:32px;font-weight:700;letter-spacing:8px;color:#14352f;margin:24px 0">%s</p>
    <p style="color:#666;font-size:13px">Kod 10 dakika geçerlidir. Bu girişi siz yapmadıysanız şifrenizi değiştirin.</p>
  </div>
</body>
</html>`, subject, code)
	return subject, plain, html
}

func DeviceLoginNoticeEmail(label, ip string) (subject, plain, html string) {
	subject = "Miyuna — yeni oturum bildirimi"
	plain = fmt.Sprintf("Hesabınıza yeni bir oturum açıldı.\nCihaz: %s\nIP: %s", label, ip)
	html = fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr"><body style="font-family:system-ui,sans-serif;padding:24px">
<p>Hesabınıza yeni bir oturum açıldı.</p>
<p><strong>Cihaz:</strong> %s<br><strong>IP:</strong> %s</p>
</body></html>`, label, ip)
	return subject, plain, html
}
