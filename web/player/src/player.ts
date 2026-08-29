import Hls from "hls.js";

interface CTA {
  id: number;
  trigger: "timestamp" | "end";
  timestampSeconds: number;
  label: string;
  url: string;
}

interface LeadGate {
  enabled: boolean;
  position: "before" | "timestamp";
  timestampSeconds: number;
  headline: string;
  requireName: boolean;
}

interface Manifest {
  id: number;
  title: string;
  playlistUrl: string;
  thumbnailUrl?: string;
  durationSeconds: number;
  ctas: CTA[];
  leadGate: LeadGate;
}

const HEARTBEAT_INTERVAL_MS = 5000;

function apiBase(): string {
  const script = document.currentScript as HTMLScriptElement | null;
  if (script?.src) {
    return new URL(script.src).origin;
  }
  // Fallback for a script loaded without currentScript support (rare, old
  // browsers): assume the player is served from the same origin as the page.
  return window.location.origin;
}

function uuid(): string {
  if (crypto.randomUUID) return crypto.randomUUID();
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

class ClipfolioEmbed {
  private base: string;
  private videoId: string;
  private sessionId: string;
  private container: HTMLElement;
  private video!: HTMLVideoElement;
  private manifest?: Manifest;
  private shownCTAIds = new Set<number>();
  private gateShown = false;
  private gateResolved = false;
  private heartbeat?: number;

  constructor(base: string, container: HTMLElement) {
    this.base = base;
    this.container = container;
    this.videoId = container.dataset.clipfolioVideo!;
    this.sessionId = uuid();
  }

  async init() {
    this.container.classList.add("clipfolio-player");
    this.container.innerHTML = `<div class="clipfolio-player__loading">Loading video…</div>`;

    let manifest: Manifest;
    try {
      const res = await fetch(`${this.base}/api/embed/${this.videoId}`);
      if (!res.ok) throw new Error(`manifest fetch failed: ${res.status}`);
      manifest = await res.json();
    } catch {
      this.container.innerHTML = `<div class="clipfolio-player__error">This video isn't available.</div>`;
      return;
    }
    this.manifest = manifest;
    this.render();
    this.reportProgress(0, false); // impression
  }

  private render() {
    const m = this.manifest!;
    this.container.innerHTML = "";
    this.injectStyles();

    const wrap = document.createElement("div");
    wrap.className = "clipfolio-player__wrap";

    this.video = document.createElement("video");
    this.video.className = "clipfolio-player__video";
    this.video.controls = true;
    this.video.playsInline = true;
    if (m.thumbnailUrl) this.video.poster = m.thumbnailUrl;

    wrap.appendChild(this.video);
    this.container.appendChild(wrap);

    this.attachSource(m.playlistUrl);
    this.wireEvents();

    const gate = m.leadGate;
    if (gate.enabled && gate.position === "before") {
      this.showLeadGate(wrap, gate);
    }
  }

  private attachSource(playlistUrl: string) {
    if (this.video.canPlayType("application/vnd.apple.mpegurl")) {
      this.video.src = playlistUrl;
      return;
    }
    if (Hls.isSupported()) {
      const hls = new Hls();
      hls.loadSource(playlistUrl);
      hls.attachMedia(this.video);
      return;
    }
    // No HLS support at all (very old browser) - nothing more we can do.
    this.video.src = playlistUrl;
  }

  private wireEvents() {
    const m = this.manifest!;

    this.video.addEventListener("play", () => {
      this.reportProgress(this.video.currentTime, true);
      this.startHeartbeat();

      const gate = m.leadGate;
      if (gate.enabled && gate.position === "before" && !this.gateResolved) {
        this.video.pause();
      }
    });

    this.video.addEventListener("pause", () => this.stopHeartbeat());

    this.video.addEventListener("timeupdate", () => {
      const gate = m.leadGate;
      if (
        gate.enabled &&
        gate.position === "timestamp" &&
        !this.gateShown &&
        !this.gateResolved &&
        this.video.currentTime >= gate.timestampSeconds
      ) {
        this.video.pause();
        this.showLeadGate(this.container.querySelector(".clipfolio-player__wrap")!, gate);
      }

      for (const cta of m.ctas) {
        if (cta.trigger !== "timestamp" || this.shownCTAIds.has(cta.id)) continue;
        if (this.video.currentTime >= cta.timestampSeconds) {
          this.showCTA(cta);
        }
      }
    });

    this.video.addEventListener("ended", () => {
      this.stopHeartbeat();
      this.reportProgress(this.video.duration || this.video.currentTime, true);
      for (const cta of m.ctas) {
        if (cta.trigger === "end" && !this.shownCTAIds.has(cta.id)) {
          this.showCTA(cta);
        }
      }
    });
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeat = window.setInterval(() => {
      if (!this.video.paused) {
        this.reportProgress(this.video.currentTime, true);
      }
    }, HEARTBEAT_INTERVAL_MS);
  }

  private stopHeartbeat() {
    if (this.heartbeat) {
      window.clearInterval(this.heartbeat);
      this.heartbeat = undefined;
    }
  }

  private reportProgress(maxTimeSeconds: number, played: boolean) {
    const body = JSON.stringify({ sessionId: this.sessionId, maxTimeSeconds, played });
    const url = `${this.base}/api/embed/${this.videoId}/progress`;
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url, new Blob([body], { type: "application/json" }));
    } else {
      fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body, keepalive: true }).catch(() => {});
    }
  }

  private showCTA(cta: CTA) {
    this.shownCTAIds.add(cta.id);
    const btn = document.createElement("a");
    btn.className = "clipfolio-player__cta";
    btn.textContent = cta.label;
    btn.href = cta.url;
    btn.target = "_blank";
    btn.rel = "noopener noreferrer";
    btn.addEventListener("click", () => {
      fetch(`${this.base}/api/embed/${this.videoId}/cta-click/${cta.id}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sessionId: this.sessionId }),
        keepalive: true,
      }).catch(() => {});
    });
    this.container.querySelector(".clipfolio-player__wrap")!.appendChild(btn);
  }

  private showLeadGate(wrap: Element, gate: LeadGate) {
    this.gateShown = true;
    const overlay = document.createElement("div");
    overlay.className = "clipfolio-player__gate";
    overlay.innerHTML = `
      <form class="clipfolio-player__gate-form">
        <p class="clipfolio-player__gate-headline"></p>
        ${gate.requireName ? '<input type="text" name="name" placeholder="Name" required />' : ""}
        <input type="email" name="email" placeholder="Email address" required />
        <button type="submit">Continue watching</button>
      </form>
    `;
    overlay.querySelector(".clipfolio-player__gate-headline")!.textContent = gate.headline;

    const form = overlay.querySelector("form")!;
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const data = new FormData(form);
      const email = String(data.get("email") || "");
      const name = String(data.get("name") || "");
      try {
        const res = await fetch(`${this.base}/api/embed/${this.videoId}/leads`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ sessionId: this.sessionId, email, name }),
        });
        if (!res.ok) throw new Error("lead submission failed");
      } catch {
        const err = overlay.querySelector(".clipfolio-player__gate-error");
        if (!err) {
          const p = document.createElement("p");
          p.className = "clipfolio-player__gate-error";
          p.textContent = "Something went wrong - please try again.";
          form.appendChild(p);
        }
        return;
      }
      this.gateResolved = true;
      overlay.remove();
      this.video.play().catch(() => {});
    });

    wrap.appendChild(overlay);
  }

  private injectStyles() {
    if (document.getElementById("clipfolio-player-styles")) return;
    const style = document.createElement("style");
    style.id = "clipfolio-player-styles";
    style.textContent = `
      .clipfolio-player__wrap { position: relative; width: 100%; background: #000; }
      .clipfolio-player__video { width: 100%; display: block; }
      .clipfolio-player__loading, .clipfolio-player__error {
        display: flex; align-items: center; justify-content: center;
        aspect-ratio: 16/9; background: #111; color: #ccc; font: 14px sans-serif;
      }
      .clipfolio-player__cta {
        position: absolute; bottom: 56px; right: 16px;
        background: #2563eb; color: #fff; padding: 10px 18px; border-radius: 6px;
        font: 600 14px/1 sans-serif; text-decoration: none; box-shadow: 0 2px 8px rgba(0,0,0,.35);
      }
      .clipfolio-player__gate {
        position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
        background: rgba(0,0,0,.75);
      }
      .clipfolio-player__gate-form {
        background: #fff; padding: 24px; border-radius: 8px; width: min(320px, 90%);
        display: flex; flex-direction: column; gap: 10px; font: 14px sans-serif;
      }
      .clipfolio-player__gate-headline { margin: 0 0 4px; font-weight: 600; color: #111; }
      .clipfolio-player__gate-form input {
        padding: 8px 10px; border: 1px solid #d1d5db; border-radius: 4px; font: inherit;
      }
      .clipfolio-player__gate-form button {
        padding: 9px; border: none; border-radius: 4px; background: #2563eb; color: #fff;
        font: 600 14px sans-serif; cursor: pointer;
      }
      .clipfolio-player__gate-error { color: #dc2626; margin: 0; font-size: 12px; }
    `;
    document.head.appendChild(style);
  }
}

function initAll(base: string) {
  document.querySelectorAll<HTMLElement>("[data-clipfolio-video]").forEach((el) => {
    if (el.dataset.clipfolioInitialized) return;
    el.dataset.clipfolioInitialized = "true";
    new ClipfolioEmbed(base, el).init();
  });
}

(function main() {
  const base = apiBase();
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => initAll(base));
  } else {
    initAll(base);
  }
})();
