package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendLead_PostsExpectedPayload(t *testing.T) {
	var received LeadPayload
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode webhook body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewSender()
	err := sender.SendLead(context.Background(), srv.URL, LeadPayload{
		Event:      "lead.captured",
		VideoID:    42,
		SessionID:  "sess-1",
		Email:      "viewer@example.com",
		CapturedAt: "2026-08-29T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("SendLead: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("expected application/json content-type, got %q", gotContentType)
	}
	if received.VideoID != 42 || received.Email != "viewer@example.com" || received.Event != "lead.captured" {
		t.Errorf("unexpected payload received: %+v", received)
	}
}

func TestSendLead_EmptyURLIsNoop(t *testing.T) {
	sender := NewSender()
	if err := sender.SendLead(context.Background(), "", LeadPayload{}); err != nil {
		t.Fatalf("expected no error for empty webhook URL, got %v", err)
	}
}

func TestSendLead_NonSuccessStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sender := NewSender()
	err := sender.SendLead(context.Background(), srv.URL, LeadPayload{Event: "lead.captured"})
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}
