import type { DropoffPoint } from "../lib/types";

const WIDTH = 640;
const HEIGHT = 200;
const PADDING = 32;

export function DropoffChart({ points }: { points: DropoffPoint[] }) {
  if (points.length === 0) {
    return <p className="empty-state">No viewers yet.</p>;
  }

  const maxTime = points[points.length - 1].timeSeconds || 1;
  const innerWidth = WIDTH - PADDING * 2;
  const innerHeight = HEIGHT - PADDING * 2;

  const coords = points.map((p) => ({
    x: PADDING + (p.timeSeconds / maxTime) * innerWidth,
    y: PADDING + (1 - p.retentionFraction) * innerHeight,
  }));

  const linePath = coords.map((c, i) => `${i === 0 ? "M" : "L"}${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(" ");
  const areaPath = `${linePath} L${coords[coords.length - 1].x.toFixed(1)},${PADDING + innerHeight} L${coords[0].x.toFixed(1)},${PADDING + innerHeight} Z`;

  return (
    <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} className="dropoff-chart" role="img" aria-label="Viewer drop-off curve">
      {[0, 0.25, 0.5, 0.75, 1].map((frac) => {
        const y = PADDING + frac * innerHeight;
        return (
          <g key={frac}>
            <line x1={PADDING} y1={y} x2={WIDTH - PADDING} y2={y} className="dropoff-chart__gridline" />
            <text x={4} y={y + 4} className="dropoff-chart__axis-label">
              {Math.round((1 - frac) * 100)}%
            </text>
          </g>
        );
      })}
      <path d={areaPath} className="dropoff-chart__area" />
      <path d={linePath} className="dropoff-chart__line" />
      <text x={PADDING} y={HEIGHT - 6} className="dropoff-chart__axis-label">
        0:00
      </text>
      <text x={WIDTH - PADDING} y={HEIGHT - 6} textAnchor="end" className="dropoff-chart__axis-label">
        {formatDuration(maxTime)}
      </text>
    </svg>
  );
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.round(seconds % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}
