import type { Video } from "../lib/types";

export function StatusBadge({ status }: { status: Video["status"] }) {
  return <span className={`status-badge status-badge--${status}`}>{status}</span>;
}
