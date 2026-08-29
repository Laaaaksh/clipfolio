import { useState, type FormEvent } from "react";
import { api } from "../lib/api";
import type { CTA } from "../lib/types";

export function CTAManager({ videoId, ctas, onChange }: { videoId: number; ctas: CTA[]; onChange: () => void }) {
  const [trigger, setTrigger] = useState<CTA["trigger"]>("end");
  const [timestamp, setTimestamp] = useState("0");
  const [label, setLabel] = useState("");
  const [url, setUrl] = useState("");
  const [busy, setBusy] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!label.trim() || !url.trim()) return;
    setBusy(true);
    try {
      await api.createCTA(videoId, {
        trigger,
        timestampSeconds: trigger === "timestamp" ? Number(timestamp) : 0,
        label: label.trim(),
        url: url.trim(),
      });
      setLabel("");
      setUrl("");
      setTimestamp("0");
      onChange();
    } finally {
      setBusy(false);
    }
  };

  const onDelete = async (ctaId: number) => {
    await api.deleteCTA(videoId, ctaId);
    onChange();
  };

  return (
    <div className="panel">
      <h3>Calls to action</h3>
      {ctas.length > 0 && (
        <ul className="cta-list">
          {ctas.map((cta) => (
            <li key={cta.id}>
              <span className="cta-trigger">
                {cta.trigger === "end" ? "At end" : `At ${cta.timestampSeconds}s`}
              </span>
              <strong>{cta.label}</strong>
              <a href={cta.url} target="_blank" rel="noreferrer">
                {cta.url}
              </a>
              <button type="button" className="link-button" onClick={() => onDelete(cta.id)}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
      <form className="cta-form" onSubmit={onSubmit}>
        <select value={trigger} onChange={(e) => setTrigger(e.target.value as CTA["trigger"])}>
          <option value="end">At video end</option>
          <option value="timestamp">At timestamp</option>
        </select>
        {trigger === "timestamp" && (
          <input
            type="number"
            min={0}
            step={1}
            value={timestamp}
            onChange={(e) => setTimestamp(e.target.value)}
            style={{ width: 80 }}
            placeholder="seconds"
          />
        )}
        <input type="text" placeholder="Button label" value={label} onChange={(e) => setLabel(e.target.value)} />
        <input type="url" placeholder="https://…" value={url} onChange={(e) => setUrl(e.target.value)} />
        <button type="submit" disabled={busy}>
          Add
        </button>
      </form>
    </div>
  );
}
