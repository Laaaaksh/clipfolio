// Package webhook delivers lead-capture events to a customer-configured URL,
// so clipfolio never needs native HubSpot/Marketo/Pardot integrations - the
// operator wires the webhook into whatever CRM they already use.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LeadPayload is the JSON body POSTed to a video's webhook URL when it
// captures a lead.
type LeadPayload struct {
	Event      string `json:"event"`
	VideoID    int64  `json:"videoId"`
	SessionID  string `json:"sessionId"`
	Email      string `json:"email"`
	Name       string `json:"name,omitempty"`
	CapturedAt string `json:"capturedAt"`
}

// Sender delivers webhook payloads over HTTP.
type Sender struct {
	client *http.Client
}

// NewSender builds a Sender with a bounded request timeout.
func NewSender() *Sender {
	return &Sender{client: &http.Client{Timeout: 10 * time.Second}}
}

// SendLead POSTs a lead-capture event to url as JSON. A non-2xx response or
// network error is returned to the caller to log; webhook delivery is
// best-effort and never blocks the viewer-facing lead submission.
func (s *Sender) SendLead(ctx context.Context, url string, payload LeadPayload) error {
	if url == "" {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
