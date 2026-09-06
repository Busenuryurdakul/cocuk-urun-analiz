package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Verifier struct {
	SecretKey string
	Enabled   bool
	Client    *http.Client
}

func NewVerifier(secretKey string, enabled bool) *Verifier {
	return &Verifier{
		SecretKey: secretKey,
		Enabled:   enabled && secretKey != "",
		Client:    &http.Client{Timeout: 10 * time.Second},
	}
}

type verifyResponse struct {
	Success bool     `json:"success"`
	Errors  []string `json:"error-codes"`
}

func (v *Verifier) Verify(ctx context.Context, token, remoteIP string) error {
	if !v.Enabled {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("turnstile token required")
	}
	form := url.Values{}
	form.Set("secret", v.SecretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := v.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var parsed verifyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	if !parsed.Success {
		return fmt.Errorf("turnstile verification failed")
	}
	return nil
}
