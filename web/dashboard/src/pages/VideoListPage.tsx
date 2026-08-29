import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import type { Video } from "../lib/types";
import { StatusBadge } from "../components/StatusBadge";

export function VideoListPage() {
  const [videos, setVideos] = useState<Video[] | null>(null);
  const [title, setTitle] = useState("");
  const [creating, setCreating] = useState(false);
  const navigate = useNavigate();

  const load = () => api.listVideos().then(setVideos);

  useEffect(() => {
    load();
    const interval = setInterval(load, 4000);
    return () => clearInterval(interval);
  }, []);

  const onCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    setCreating(true);
    try {
      const video = await api.createVideo(title.trim());
      setTitle("");
      navigate(`/videos/${video.id}`);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="page">
      <h2>Videos</h2>

      <form className="inline-form" onSubmit={onCreate}>
        <input
          type="text"
          placeholder="New video title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <button type="submit" disabled={creating}>
          Create
        </button>
      </form>

      {videos === null ? (
        <p>Loading…</p>
      ) : videos.length === 0 ? (
        <p className="empty-state">No videos yet. Create one above, then upload a file.</p>
      ) : (
        <table className="video-table">
          <thead>
            <tr>
              <th>Title</th>
              <th>Status</th>
              <th>Duration</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {videos.map((v) => (
              <tr key={v.id}>
                <td>
                  <Link to={`/videos/${v.id}`}>{v.title}</Link>
                </td>
                <td>
                  <StatusBadge status={v.status} />
                </td>
                <td>{v.durationSeconds ? `${Math.round(v.durationSeconds)}s` : "—"}</td>
                <td>{new Date(v.createdAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
