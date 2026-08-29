export interface Video {
  id: number;
  ownerId: number;
  title: string;
  status: "uploading" | "transcoding" | "ready" | "failed";
  error: string;
  durationSeconds: number;
  sourceKey: string;
  playlistKey: string;
  thumbnailKey: string;
  webhookUrl: string;
  createdAt: string;
  updatedAt: string;
}

export interface CTA {
  id: number;
  videoId: number;
  trigger: "timestamp" | "end";
  timestampSeconds: number;
  label: string;
  url: string;
  createdAt: string;
}

export interface LeadGate {
  videoId: number;
  enabled: boolean;
  position: "before" | "timestamp";
  timestampSeconds: number;
  headline: string;
  requireName: boolean;
}

export interface Lead {
  id: number;
  videoId: number;
  sessionId: string;
  email: string;
  name: string;
  createdAt: string;
}

export interface DropoffPoint {
  timeSeconds: number;
  retentionFraction: number;
}

export interface AnalyticsSummary {
  impressions: number;
  plays: number;
  playRate: number;
  avgWatchPercentage: number;
  dropoffCurve: DropoffPoint[];
}
