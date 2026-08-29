import { useState, type FormEvent } from "react";
import { api } from "../lib/api";
import type { LeadGate } from "../lib/types";

export function LeadGateEditor({
  videoId,
  gate: initial,
  onChange,
}: {
  videoId: number;
  gate: LeadGate;
  onChange: (gate: LeadGate) => void;
}) {
  const [gate, setGate] = useState(initial);
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      const updated = await api.setLeadGate(videoId, {
        enabled: gate.enabled,
        position: gate.position,
        timestampSeconds: gate.timestampSeconds,
        headline: gate.headline,
        requireName: gate.requireName,
      });
      const withVideoId = { ...updated, videoId };
      setGate(withVideoId);
      onChange(withVideoId);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="panel">
      <h3>Lead-capture gate</h3>
      <form className="lead-gate-form" onSubmit={onSubmit}>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={gate.enabled}
            onChange={(e) => setGate({ ...gate, enabled: e.target.checked })}
          />
          Require an email before {gate.position === "before" ? "playback starts" : "playback continues"}
        </label>

        {gate.enabled && (
          <>
            <label>
              Show
              <select
                value={gate.position}
                onChange={(e) => setGate({ ...gate, position: e.target.value as LeadGate["position"] })}
              >
                <option value="before">Before playback starts</option>
                <option value="timestamp">At a timestamp</option>
              </select>
            </label>
            {gate.position === "timestamp" && (
              <label>
                Timestamp (seconds)
                <input
                  type="number"
                  min={0}
                  value={gate.timestampSeconds}
                  onChange={(e) => setGate({ ...gate, timestampSeconds: Number(e.target.value) })}
                />
              </label>
            )}
            <label>
              Headline
              <input
                type="text"
                value={gate.headline}
                onChange={(e) => setGate({ ...gate, headline: e.target.value })}
              />
            </label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={gate.requireName}
                onChange={(e) => setGate({ ...gate, requireName: e.target.checked })}
              />
              Also require a name
            </label>
          </>
        )}

        <button type="submit" disabled={busy}>
          Save
        </button>
        {saved && <span className="save-confirmation">Saved</span>}
      </form>
    </div>
  );
}
