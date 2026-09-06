package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ResendConfig struct {
	APIKey string
	From   string
}

type ResendService struct {
	cfg    ResendConfig
	client *http.Client
}

func NewResend(cfg ResendConfig) *ResendService {
	return &ResendService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *ResendService) Send(ctx context.Context, msg Message) error {
	if strings.TrimSpace(s.cfg.APIKey) == "" {
		return fmt.Errorf("resend api key missing")
	}
	from, _ := parseFrom(s.cfg.From)
	body := map[string]any{
		"from":    from,
		"to":      []string{msg.To},
		"subject": msg.Subject,
	}
	if msg.HTMLBody != "" {
		body["html"] = msg.HTMLBody
		body["text"] = msg.Body
	} else {
		body["text"] = msg.Body
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend api %d: %s", resp.StatusCode, strings.TrimSpace(string(slurp)))
	}
	return nil
}
