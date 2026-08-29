package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/laaaaksh/clipfolio/internal/analytics"
	"github.com/laaaaksh/clipfolio/internal/auth"
	"github.com/laaaaksh/clipfolio/internal/config"
	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/testutil"
)

// newTestClient builds an http.Client with a cookie jar so a login's session
// cookie carries into subsequent requests, matching how a browser behaves.
func newTestClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	return &http.Client{Jar: jar}
}

func newTestServer(t *testing.T) (*httptest.Server, *db.Store) {
	t.Helper()
	store := testutil.OpenTestStore(t)

	server := NewServer(store, newFakeObjectStore(), config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	server.Start(ctx, 2)

	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)
	return ts, store
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(mustJSON(t, body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func id(n int64) string {
	return strconv.FormatInt(n, 10)
}

const adminEmail = "admin@example.com"
const adminPassword = "hunter22222"

func setupAccount(t *testing.T, ts *httptest.Server, client *http.Client) {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/setup", map[string]string{
		"email":    adminEmail,
		"password": adminPassword,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("setup: expected 201, got %d: %s", resp.StatusCode, body)
	}
}

func TestSetupStatus_ReflectsWhetherAnAccountExists(t *testing.T) {
	ts, _ := newTestServer(t)
	client := newTestClient(t)

	before := doJSON(t, client, http.MethodGet, ts.URL+"/api/setup", nil)
	var beforeBody map[string]bool
	decodeBody(t, before, &beforeBody)
	if !beforeBody["needsSetup"] {
		t.Fatalf("expected needsSetup=true before any account exists, got %+v", beforeBody)
	}

	setupAccount(t, ts, client)

	after := doJSON(t, client, http.MethodGet, ts.URL+"/api/setup", nil)
	var afterBody map[string]bool
	decodeBody(t, after, &afterBody)
	if afterBody["needsSetup"] {
		t.Fatalf("expected needsSetup=false after setup, got %+v", afterBody)
	}
}

func TestSetup_OnlyWorksOnce(t *testing.T) {
	ts, _ := newTestServer(t)
	client := newTestClient(t)

	setupAccount(t, ts, client)

	second := doJSON(t, client, http.MethodPost, ts.URL+"/api/setup", map[string]string{
		"email":    "someone-else@example.com",
		"password": "irrelevant1",
	})
	defer second.Body.Close()
	if second.StatusCode != http.StatusForbidden {
		t.Fatalf("expected second setup call to be forbidden, got %d", second.StatusCode)
	}
}

func TestLogin_WrongPasswordRejected(t *testing.T) {
	ts, _ := newTestServer(t)
	setupAccount(t, ts, newTestClient(t))

	client := newTestClient(t)
	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/login", map[string]string{
		"email":    adminEmail,
		"password": "wrong-password",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", resp.StatusCode)
	}
}

func TestVideosEndpoint_RequiresAuth(t *testing.T) {
	ts, _ := newTestServer(t)
	client := newTestClient(t)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/videos/", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", resp.StatusCode)
	}
}

func TestVideos_CrossAccountIsolation(t *testing.T) {
	ts, store := newTestServer(t)

	ownerClient := newTestClient(t)
	setupAccount(t, ts, ownerClient)

	var video db.Video
	createResp := doJSON(t, ownerClient, http.MethodPost, ts.URL+"/api/videos/", map[string]string{"title": "Owner's video"})
	decodeBody(t, createResp, &video)

	// A second account (created directly, bypassing /api/setup which only
	// runs once) must not be able to fetch the first account's video.
	otherHash, err := auth.HashPassword("otherPassword1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := store.CreateUser(context.Background(), "other@example.com", otherHash); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	otherClient := newTestClient(t)
	loginResp := doJSON(t, otherClient, http.MethodPost, ts.URL+"/api/login", map[string]string{
		"email":    "other@example.com",
		"password": "otherPassword1",
	})
	loginResp.Body.Close()

	getResp := doJSON(t, otherClient, http.MethodGet, ts.URL+"/api/videos/"+id(video.ID), nil)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 fetching another account's video, got %d", getResp.StatusCode)
	}
}

// TestListEndpoints_EmptyListsAreJSONArraysNotNull guards against a real bug:
// a nil Go slice marshals to JSON `null`, and the dashboard crashes calling
// `.length` on that. Every list endpoint must return `[]`, never `null`,
// when there are zero rows.
func TestListEndpoints_EmptyListsAreJSONArraysNotNull(t *testing.T) {
	ts, _ := newTestServer(t)
	client := newTestClient(t)
	setupAccount(t, ts, client)

	var video db.Video
	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/videos/", map[string]string{"title": "Empty lists"})
	decodeBody(t, createResp, &video)

	cases := []struct {
		name string
		url  string
	}{
		{"videos list", ts.URL + "/api/videos/"},
		{"ctas list", ts.URL + "/api/videos/" + id(video.ID) + "/ctas/"},
		{"leads list", ts.URL + "/api/videos/" + id(video.ID) + "/leads"},
	}
	// Reset videos list to only contain videos with zero children by checking
	// raw body text for a literal "null" rather than decoding, since decoding
	// into []T would silently accept null as a nil slice.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, client, http.MethodGet, tc.url, nil)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			trimmed := string(bytes.TrimSpace(body))
			if trimmed == "null" {
				t.Fatalf("%s returned JSON null instead of an empty array", tc.name)
			}
		})
	}
}

// TestJSONFieldsAreCamelCase guards against a real bug: the Go db structs
// have no json tags themselves, and embedding them in an API response
// without tags silently serializes fields as PascalCase (Go's default),
// which every JS/TS consumer (the dashboard, the embeddable player) expects
// as camelCase. A previous version of this API shipped with a broken
// embeddable player because of exactly this - CTAs and the lead gate never
// fired because `cta.trigger`/`gate.enabled` were `undefined` in the
// browser, since the wire format was actually `Trigger`/`Enabled`.
func TestJSONFieldsAreCamelCase(t *testing.T) {
	ts, _ := newTestServer(t)
	client := newTestClient(t)
	setupAccount(t, ts, client)

	var video db.Video
	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/videos/", map[string]string{"title": "Casing check"})
	decodeBody(t, createResp, &video)

	ctaResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/videos/"+id(video.ID)+"/ctas/", map[string]any{
		"trigger": "end", "label": "Go", "url": "https://example.com",
	})
	ctaBody, _ := io.ReadAll(ctaResp.Body)
	ctaResp.Body.Close()
	for _, want := range []string{`"trigger"`, `"timestampSeconds"`, `"label"`, `"url"`, `"videoId"`} {
		if !bytes.Contains(ctaBody, []byte(want)) {
			t.Errorf("CTA JSON missing expected camelCase field %s: %s", want, ctaBody)
		}
	}
	for _, unwanted := range []string{`"Trigger"`, `"TimestampSeconds"`, `"Label"`, `"URL"`, `"VideoID"`} {
		if bytes.Contains(ctaBody, []byte(unwanted)) {
			t.Errorf("CTA JSON contains PascalCase field %s, expected camelCase: %s", unwanted, ctaBody)
		}
	}

	gateResp := doJSON(t, client, http.MethodPut, ts.URL+"/api/videos/"+id(video.ID)+"/lead-gate/", map[string]any{
		"enabled": true, "position": "timestamp", "timestampSeconds": 5, "headline": "Hi", "requireName": false,
	})
	gateBody, _ := io.ReadAll(gateResp.Body)
	gateResp.Body.Close()
	for _, want := range []string{`"enabled"`, `"position"`, `"timestampSeconds"`, `"headline"`, `"requireName"`} {
		if !bytes.Contains(gateBody, []byte(want)) {
			t.Errorf("LeadGate JSON missing expected camelCase field %s: %s", want, gateBody)
		}
	}
	for _, unwanted := range []string{`"Enabled"`, `"Position"`, `"TimestampSeconds"`, `"Headline"`, `"RequireName"`} {
		if bytes.Contains(gateBody, []byte(unwanted)) {
			t.Errorf("LeadGate JSON contains PascalCase field %s, expected camelCase: %s", unwanted, gateBody)
		}
	}
}

func TestFullVideoLifecycle_UploadTranscodeAnalyticsLeadsCTAs(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	ts, _ := newTestServer(t)
	client := newTestClient(t)
	setupAccount(t, ts, client)

	// 1. Create the video record.
	var video db.Video
	createResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/videos/", map[string]string{"title": "Product demo"})
	decodeBody(t, createResp, &video)
	if video.Status != db.VideoStatusUploading {
		t.Fatalf("expected initial status uploading, got %s", video.Status)
	}

	// 2. Configure a webhook target to receive the lead we'll submit later.
	receivedWebhook := make(chan map[string]any, 1)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		receivedWebhook <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	webhookSetResp := doJSON(t, client, http.MethodPut, ts.URL+"/api/videos/"+id(video.ID)+"/webhook", map[string]string{"url": webhookServer.URL})
	webhookSetResp.Body.Close()
	if webhookSetResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 setting webhook, got %d", webhookSetResp.StatusCode)
	}

	// 3. Add a CTA.
	ctaResp := doJSON(t, client, http.MethodPost, ts.URL+"/api/videos/"+id(video.ID)+"/ctas/", map[string]any{
		"trigger": "end", "label": "Book a demo", "url": "https://example.com/demo",
	})
	ctaResp.Body.Close()
	if ctaResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating CTA, got %d", ctaResp.StatusCode)
	}

	// 4. Upload a real, tiny synthetic clip and wait for it to reach "ready".
	uploadTestClip(t, client, ts.URL, video.ID)
	waitForVideoStatus(t, client, ts.URL, video.ID, db.VideoStatusReady, 30*time.Second)

	// 5. The public embed manifest should now describe a playable video.
	embedResp, err := http.Get(ts.URL + "/api/embed/" + id(video.ID))
	if err != nil {
		t.Fatalf("GET embed manifest: %v", err)
	}
	var manifest struct {
		PlaylistURL     string  `json:"playlistUrl"`
		DurationSeconds float64 `json:"durationSeconds"`
	}
	decodeBody(t, embedResp, &manifest)
	if manifest.PlaylistURL == "" || manifest.DurationSeconds <= 0 {
		t.Fatalf("expected a playable manifest, got %+v", manifest)
	}

	// 6. Simulate a viewer: report watch progress, then submit a lead.
	progressResp, err := http.Post(ts.URL+"/api/embed/"+id(video.ID)+"/progress", "application/json",
		bytes.NewReader(mustJSON(t, map[string]any{"sessionId": "viewer-1", "maxTimeSeconds": manifest.DurationSeconds / 2, "played": true})))
	if err != nil {
		t.Fatalf("POST progress: %v", err)
	}
	progressResp.Body.Close()

	leadResp, err := http.Post(ts.URL+"/api/embed/"+id(video.ID)+"/leads", "application/json",
		bytes.NewReader(mustJSON(t, map[string]any{"sessionId": "viewer-1", "email": "viewer@example.com", "name": "Viewer"})))
	if err != nil {
		t.Fatalf("POST lead: %v", err)
	}
	if leadResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating lead, got %d", leadResp.StatusCode)
	}
	leadResp.Body.Close()

	select {
	case payload := <-receivedWebhook:
		if payload["email"] != "viewer@example.com" {
			t.Fatalf("expected webhook to carry the lead's email, got %+v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the lead webhook to fire")
	}

	// 7. Analytics should now reflect the one played, half-watched session.
	analyticsResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/videos/"+id(video.ID)+"/analytics", nil)
	var summary analytics.Summary
	decodeBody(t, analyticsResp, &summary)
	if summary.Plays != 1 {
		t.Fatalf("expected 1 play recorded, got %d", summary.Plays)
	}
	if summary.AvgWatchPercentage < 0.4 || summary.AvgWatchPercentage > 0.6 {
		t.Fatalf("expected ~50%% avg watch percentage, got %v", summary.AvgWatchPercentage)
	}

	// 8. The leads list should show the captured lead.
	leadsResp := doJSON(t, client, http.MethodGet, ts.URL+"/api/videos/"+id(video.ID)+"/leads", nil)
	var leads []db.Lead
	decodeBody(t, leadsResp, &leads)
	if len(leads) != 1 || leads[0].Email != "viewer@example.com" {
		t.Fatalf("expected one lead for viewer@example.com, got %+v", leads)
	}
}

func uploadTestClip(t *testing.T, client *http.Client, baseURL string, videoID int64) {
	t.Helper()

	dir := t.TempDir()
	clipPath := dir + "/clip.mp4"
	gen := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-c:v", "libx264", "-c:a", "aac", "-shortest", clipPath,
	)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate test clip: %v\n%s", err, out)
	}

	clipData, err := os.ReadFile(clipPath)
	if err != nil {
		t.Fatalf("read generated clip: %v", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "clip.mp4")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(clipData); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/videos/"+id(videoID)+"/upload", &buf)
	if err != nil {
		t.Fatalf("new upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 202 accepted for upload, got %d: %s", resp.StatusCode, body)
	}
}

func waitForVideoStatus(t *testing.T, client *http.Client, baseURL string, videoID int64, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := doJSON(t, client, http.MethodGet, baseURL+"/api/videos/"+id(videoID), nil)
		var v db.Video
		decodeBody(t, resp, &v)
		if v.Status == want {
			return
		}
		if v.Status == db.VideoStatusFailed {
			t.Fatalf("video transcoding failed: %s", v.Error)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for video %d to reach status %q", videoID, want)
}
