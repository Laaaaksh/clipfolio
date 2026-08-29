package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/testutil"
)

func TestUsers_CreateAndFetch(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, err := store.CreateUser(ctx, "admin@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	byEmail, err := store.GetUserByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Fatalf("expected id %d, got %d", u.ID, byEmail.ID)
	}

	byID, err := store.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if byID.Email != "admin@example.com" {
		t.Fatalf("expected email admin@example.com, got %s", byID.Email)
	}
}

func TestUsers_DuplicateEmailRejected(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	if _, err := store.CreateUser(ctx, "dup@example.com", "hash"); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	if _, err := store.CreateUser(ctx, "dup@example.com", "hash2"); err == nil {
		t.Fatal("expected duplicate email to be rejected by the unique constraint")
	}
}

func TestSessions_ExpiredSessionNotReturned(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, err := store.CreateUser(ctx, "expiring@example.com", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := store.CreateSession(ctx, "live-token", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession (live): %v", err)
	}
	if err := store.CreateSession(ctx, "expired-token", u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("CreateSession (expired): %v", err)
	}

	if _, err := store.GetSession(ctx, "live-token"); err != nil {
		t.Fatalf("expected live session to be returned, got %v", err)
	}
	if _, err := store.GetSession(ctx, "expired-token"); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for expired session, got %v", err)
	}
}

func TestVideos_LifecycleStatusTransitions(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "owner@example.com", "hash")
	v, err := store.CreateVideo(ctx, u.ID, "Demo video")
	if err != nil {
		t.Fatalf("CreateVideo: %v", err)
	}
	if v.Status != db.VideoStatusUploading {
		t.Fatalf("expected initial status %q, got %q", db.VideoStatusUploading, v.Status)
	}

	if err := store.SetVideoSourceKey(ctx, v.ID, "videos/1/source.mp4"); err != nil {
		t.Fatalf("SetVideoSourceKey: %v", err)
	}
	updated, _ := store.GetVideo(ctx, v.ID)
	if updated.Status != db.VideoStatusTranscoding {
		t.Fatalf("expected status %q after upload, got %q", db.VideoStatusTranscoding, updated.Status)
	}

	if err := store.MarkVideoReady(ctx, v.ID, 42.5, "videos/1/master.m3u8", "videos/1/thumbnail.jpg"); err != nil {
		t.Fatalf("MarkVideoReady: %v", err)
	}
	ready, _ := store.GetVideo(ctx, v.ID)
	if ready.Status != db.VideoStatusReady || ready.DurationSeconds != 42.5 || ready.PlaylistKey != "videos/1/master.m3u8" {
		t.Fatalf("unexpected video after MarkVideoReady: %+v", ready)
	}
}

func TestVideos_MarkFailedRecordsError(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "owner2@example.com", "hash")
	v, _ := store.CreateVideo(ctx, u.ID, "Bad video")

	if err := store.MarkVideoFailed(ctx, v.ID, "ffmpeg exit status 1"); err != nil {
		t.Fatalf("MarkVideoFailed: %v", err)
	}
	failed, _ := store.GetVideo(ctx, v.ID)
	if failed.Status != db.VideoStatusFailed || failed.Error != "ffmpeg exit status 1" {
		t.Fatalf("unexpected video after MarkVideoFailed: %+v", failed)
	}
}

func TestViewerSessions_UpsertNeverGoesBackward(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "viewer-owner@example.com", "hash")
	v, _ := store.CreateVideo(ctx, u.ID, "Watched video")

	if err := store.UpsertViewerProgress(ctx, "sess-1", v.ID, 30, true); err != nil {
		t.Fatalf("UpsertViewerProgress (30s): %v", err)
	}
	// A late/out-of-order heartbeat reporting an earlier time (e.g. a seek
	// back, or a race between two in-flight requests) must not erase progress.
	if err := store.UpsertViewerProgress(ctx, "sess-1", v.ID, 10, false); err != nil {
		t.Fatalf("UpsertViewerProgress (10s): %v", err)
	}

	sessions, err := store.ViewerSessionsForVideo(ctx, v.ID)
	if err != nil {
		t.Fatalf("ViewerSessionsForVideo: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected exactly one viewer session (upsert, not insert), got %d", len(sessions))
	}
	if sessions[0].MaxTimeSeconds != 30 {
		t.Fatalf("expected max_time_seconds to stay at 30 (not regress to 10), got %v", sessions[0].MaxTimeSeconds)
	}
	if !sessions[0].Played {
		t.Fatalf("expected played to latch true once set, even after an update with played=false")
	}
}

func TestCTAs_DeleteScopedToVideo(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "cta-owner@example.com", "hash")
	v1, _ := store.CreateVideo(ctx, u.ID, "Video 1")
	v2, _ := store.CreateVideo(ctx, u.ID, "Video 2")

	cta, err := store.CreateCTA(ctx, v1.ID, db.CTATriggerEnd, 0, "Book a demo", "https://example.com")
	if err != nil {
		t.Fatalf("CreateCTA: %v", err)
	}

	// Deleting via the wrong video id must fail, not silently delete a CTA
	// that belongs to a video the caller doesn't own.
	if err := store.DeleteCTA(ctx, v2.ID, cta.ID); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting a CTA via the wrong video id, got %v", err)
	}

	remaining, err := store.ListCTAs(ctx, v1.ID)
	if err != nil || len(remaining) != 1 {
		t.Fatalf("expected the CTA to survive a delete attempt via the wrong video, got %+v, err=%v", remaining, err)
	}

	if err := store.DeleteCTA(ctx, v1.ID, cta.ID); err != nil {
		t.Fatalf("expected delete via the correct video id to succeed, got %v", err)
	}
}

func TestLeadGate_UpsertOverwrites(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "gate-owner@example.com", "hash")
	v, _ := store.CreateVideo(ctx, u.ID, "Gated video")

	err := store.UpsertLeadGate(ctx, db.LeadGate{VideoID: v.ID, Enabled: true, Position: db.LeadGatePositionBefore, Headline: "Sign up"})
	if err != nil {
		t.Fatalf("UpsertLeadGate (first): %v", err)
	}

	err = store.UpsertLeadGate(ctx, db.LeadGate{VideoID: v.ID, Enabled: true, Position: db.LeadGatePositionTimestamp, TimestampSeconds: 15, Headline: "Keep watching"})
	if err != nil {
		t.Fatalf("UpsertLeadGate (second): %v", err)
	}

	gate, err := store.GetLeadGate(ctx, v.ID)
	if err != nil {
		t.Fatalf("GetLeadGate: %v", err)
	}
	if gate.Position != db.LeadGatePositionTimestamp || gate.TimestampSeconds != 15 || gate.Headline != "Keep watching" {
		t.Fatalf("expected the second upsert to overwrite the first, got %+v", gate)
	}
}

func TestLeads_CreateAndList(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "lead-owner@example.com", "hash")
	v, _ := store.CreateVideo(ctx, u.ID, "Lead video")

	if _, err := store.CreateLead(ctx, v.ID, "sess-a", "a@example.com", "Alice"); err != nil {
		t.Fatalf("CreateLead: %v", err)
	}
	if _, err := store.CreateLead(ctx, v.ID, "sess-b", "b@example.com", ""); err != nil {
		t.Fatalf("CreateLead: %v", err)
	}

	leads, err := store.ListLeads(ctx, v.ID)
	if err != nil {
		t.Fatalf("ListLeads: %v", err)
	}
	if len(leads) != 2 {
		t.Fatalf("expected 2 leads, got %d", len(leads))
	}
}

func TestVideoDeletion_CascadesToRelatedRows(t *testing.T) {
	store := testutil.OpenTestStore(t)
	ctx := context.Background()

	u, _ := store.CreateUser(ctx, "cascade-owner@example.com", "hash")
	v, _ := store.CreateVideo(ctx, u.ID, "Cascade video")
	if _, err := store.CreateCTA(ctx, v.ID, db.CTATriggerEnd, 0, "CTA", "https://example.com"); err != nil {
		t.Fatalf("CreateCTA: %v", err)
	}
	if _, err := store.CreateLead(ctx, v.ID, "sess-c", "c@example.com", ""); err != nil {
		t.Fatalf("CreateLead: %v", err)
	}

	if err := store.DeleteVideo(ctx, v.ID); err != nil {
		t.Fatalf("DeleteVideo: %v", err)
	}

	if _, err := store.GetVideo(ctx, v.ID); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected video to be gone, got %v", err)
	}
	ctas, err := store.ListCTAs(ctx, v.ID)
	if err != nil || len(ctas) != 0 {
		t.Fatalf("expected CTAs to cascade-delete, got %+v err=%v", ctas, err)
	}
}
