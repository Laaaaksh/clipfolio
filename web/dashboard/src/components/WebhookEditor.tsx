import { useState, type FormEvent } from "react";
import { api } from "../lib/api";

export function WebhookEditor({ videoId, initialURL }: { videoId: number; initialURL: string }) {
  const [url, setURL] = useState(initialURL);
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await api.setWebhook(videoId, url.trim());
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="panel">
      <h3>Lead webhook</h3>
      <p className="panel-hint">
        Captured leads are POSTed here as JSON, so you can wire them into your own CRM.
      </p>
      <form className="inline-form" onSubmit={onSubmit}>
        <input
          type="url"
          placeholder="https://your-crm.example.com/webhooks/clipfolio"
          value={url}
          onChange={(e) => setURL(e.target.value)}
        />
        <button type="submit" disabled={busy}>
          Save
        </button>
        {saved && <span className="save-confirmation">Saved</span>}
      </form>
    </div>
  );
}
