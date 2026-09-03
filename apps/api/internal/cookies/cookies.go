package cookies

import (
	"net/http"
	"time"
)

const (
	SessionCookie = "miyuna_session"
	PendingCookie = "miyuna_pending"
	SetupCookie   = "miyuna_mfa_setup"
)

type Options struct {
	Secure   bool
	SameSite http.SameSite
}

func DefaultOptions() Options {
	return Options{Secure: false, SameSite: http.SameSiteLaxMode}
}

func Set(w http.ResponseWriter, name, value string, expires time.Time, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
		Expires:  expires,
	})
}

func Clear(w http.ResponseWriter, name string, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
		MaxAge:   -1,
	})
}

func Get(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}
