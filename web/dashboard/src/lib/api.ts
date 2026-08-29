import type { AnalyticsSummary, CTA, Lead, LeadGate, Video } from "./types";

class APIError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: "same-origin",
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = await res.json();
      if (data?.error) message = data.error;
    } catch {
      // ignore non-JSON error bodies
    }
    throw new APIError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  setupStatus: () => request<{ needsSetup: boolean }>("GET", "/setup"),
  setup: (email: string, password: string, setupToken: string) =>
    request<{ id: number; email: string }>("POST", "/setup", { email, password, setupToken }),
  login: (email: string, password: string) =>
    request<{ id: number; email: string }>("POST", "/login", { email, password }),
  logout: () => request<void>("POST", "/logout"),
  me: () => request<{ id: number; email: string }>("GET", "/me"),

  listVideos: () => request<Video[]>("GET", "/videos/"),
  createVideo: (title: string) => request<Video>("POST", "/videos/", { title }),
  getVideo: (id: number) => request<Video>("GET", `/videos/${id}`),
  deleteVideo: (id: number) => request<void>("DELETE", `/videos/${id}`),
  uploadVideo: (id: number, file: File, onProgress?: (pct: number) => void) =>
    uploadVideoFile(id, file, onProgress),

  videoAnalytics: (id: number) => request<AnalyticsSummary>("GET", `/videos/${id}/analytics`),
  videoLeads: (id: number) => request<Lead[]>("GET", `/videos/${id}/leads`),
  setWebhook: (id: number, url: string) => request<void>("PUT", `/videos/${id}/webhook`, { url }),

  listCTAs: (id: number) => request<CTA[]>("GET", `/videos/${id}/ctas/`),
  createCTA: (id: number, cta: Pick<CTA, "trigger" | "timestampSeconds" | "label" | "url">) =>
    request<CTA>("POST", `/videos/${id}/ctas/`, cta),
  deleteCTA: (videoId: number, ctaId: number) => request<void>("DELETE", `/videos/${videoId}/ctas/${ctaId}`),

  getLeadGate: (id: number) => request<LeadGate>("GET", `/videos/${id}/lead-gate/`),
  setLeadGate: (id: number, gate: Omit<LeadGate, "videoId">) =>
    request<LeadGate>("PUT", `/videos/${id}/lead-gate/`, gate),
};

function uploadVideoFile(id: number, file: File, onProgress?: (pct: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `/api/videos/${id}/upload`);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100));
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) resolve();
      else reject(new APIError(xhr.status, xhr.responseText || "upload failed"));
    };
    xhr.onerror = () => reject(new APIError(0, "network error during upload"));
    const form = new FormData();
    form.append("file", file);
    xhr.send(form);
  });
}

export { APIError };
