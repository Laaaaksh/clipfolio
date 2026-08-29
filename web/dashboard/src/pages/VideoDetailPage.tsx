import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../lib/api";
import type { AnalyticsSummary, CTA, Lead, LeadGate, Video } from "../lib/types";
import { StatusBadge } from "../components/StatusBadge";
import { UploadWidget } from "../components/UploadWidget";
import { EmbedPreview } from "../components/EmbedPreview";
import { DropoffChart } from "../components/DropoffChart";
import { CTAManager } from "../components/CTAManager";
import { LeadGateEditor } from "../components/LeadGateEditor";
import { LeadsTable } from "../components/LeadsTable";
import { WebhookEditor } from "../components/WebhookEditor";

export function VideoDetailPage() {
  const { id } = useParams();
  const videoId = Number(id);
  const navigate = useNavigate();

  const [video, setVideo] = useState<Video | null>(null);
  const [ctas, setCTAs] = useState<CTA[]>([]);
  const [leadGate, setLeadGate] = useState<LeadGate | null>(null);
  const [leads, setLeads] = useState<Lead[]>([]);
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null);
  const [copied, setCopied] = useState(false);

  const loadVideo = useCallback(() => api.getVideo(videoId).then(setVideo), [videoId]);

  const loadReadyData = useCallback(() => {
    api.listCTAs(videoId).then(setCTAs);
    api.getLeadGate(videoId).then(setLeadGate);
    api.videoLeads(videoId).then(setLeads);
    api.videoAnalytics(videoId).then(setAnalytics);
  }, [videoId]);

  useEffect(() => {
    loadVideo();
  }, [loadVideo]);

  useEffect(() => {
    if (video?.status === "ready") loadReadyData();
  }, [video?.status, loadReadyData]);

  useEffect(() => {
    if (video && (video.status === "ready" || video.status === "failed")) return;
    const interval = setInterval(loadVideo, 2500);
    return () => clearInterval(interval);
  }, [video, loadVideo]);

  useEffect(() => {
    if (video?.status !== "ready") return;
    const interval = setInterval(() => {
      api.videoAnalytics(videoId).then(setAnalytics);
      api.videoLeads(videoId).then(setLeads);
    }, 5000);
    return () => clearInterval(interval);
  }, [video?.status, videoId]);

  const onDelete = async () => {
    if (!confirm("Delete this video and all its data? This can't be undone.")) return;
    await api.deleteVideo(videoId);
    navigate("/");
  };

  if (!video) return <div className="page">Loading…</div>;

  const embedSnippet = `<div data-clipfolio-video="${video.id}"></div>\n<script src="${window.location.origin}/player.js" async></script>`;

  return (
    <div className="page">
      <p>
        <Link to="/">&larr; All videos</Link>
      </p>
      <div className="page-header">
        <h2>{video.title}</h2>
        <StatusBadge status={video.status} />
      </div>

      {video.status === "uploading" && <UploadWidget videoId={video.id} onUploaded={loadVideo} />}

      {video.status === "transcoding" && (
        <div className="panel">
          <p>Transcoding to adaptive-bitrate HLS… this page updates automatically.</p>
        </div>
      )}

      {video.status === "failed" && (
        <div className="panel panel--error">
          <p>Transcoding failed: {video.error}</p>
        </div>
      )}

      {video.status === "ready" && (
        <>
          <div className="panel">
            <h3>Preview</h3>
            <p className="panel-hint">Reflects saved CTA and lead-gate settings; re-renders when they change.</p>
            <EmbedPreview
              key={`${video.id}-${ctas.length}-${leadGate?.enabled}-${leadGate?.position}-${leadGate?.timestampSeconds}`}
              videoId={video.id}
            />
          </div>

          <div className="panel">
            <h3>Embed snippet</h3>
            <pre className="code-block">{embedSnippet}</pre>
            <button
              type="button"
              onClick={() => {
                navigator.clipboard.writeText(embedSnippet);
                setCopied(true);
                setTimeout(() => setCopied(false), 1500);
              }}
            >
              {copied ? "Copied!" : "Copy snippet"}
            </button>
          </div>

          {analytics && (
            <div className="panel">
              <h3>Analytics</h3>
              <div className="stat-row">
                <div className="stat">
                  <span className="stat-value">{analytics.impressions}</span>
                  <span className="stat-label">Impressions</span>
                </div>
                <div className="stat">
                  <span className="stat-value">{analytics.plays}</span>
                  <span className="stat-label">Plays</span>
                </div>
                <div className="stat">
                  <span className="stat-value">{Math.round(analytics.playRate * 100)}%</span>
                  <span className="stat-label">Play rate</span>
                </div>
                <div className="stat">
                  <span className="stat-value">{Math.round(analytics.avgWatchPercentage * 100)}%</span>
                  <span className="stat-label">Avg. watched</span>
                </div>
              </div>
              <DropoffChart points={analytics.dropoffCurve} />
            </div>
          )}

          <CTAManager videoId={video.id} ctas={ctas} onChange={() => api.listCTAs(video.id).then(setCTAs)} />

          {leadGate && <LeadGateEditor videoId={video.id} gate={leadGate} onChange={setLeadGate} />}

          <div className="panel">
            <h3>Leads</h3>
            <LeadsTable leads={leads} />
          </div>

          <WebhookEditor videoId={video.id} initialURL={video.webhookUrl} />
        </>
      )}

      <div className="panel panel--danger">
        <button type="button" className="danger-button" onClick={onDelete}>
          Delete video
        </button>
      </div>
    </div>
  );
}
