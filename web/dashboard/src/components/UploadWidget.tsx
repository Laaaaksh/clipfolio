import { useRef, useState } from "react";
import { api } from "../lib/api";

export function UploadWidget({ videoId, onUploaded }: { videoId: number; onUploaded: () => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const onFileChosen = async () => {
    const file = inputRef.current?.files?.[0];
    if (!file) return;
    setError(null);
    setProgress(0);
    try {
      await api.uploadVideo(videoId, file, setProgress);
      onUploaded();
    } catch {
      setError("Upload failed. Please try again.");
    } finally {
      setProgress(null);
    }
  };

  return (
    <div className="panel">
      <h3>Upload</h3>
      <p className="panel-hint">MP4, MOV, or WebM. Transcoding to adaptive-bitrate HLS starts automatically.</p>
      <input ref={inputRef} type="file" accept="video/*" onChange={onFileChosen} />
      {progress !== null && (
        <div className="progress-bar">
          <div className="progress-bar__fill" style={{ width: `${progress}%` }} />
        </div>
      )}
      {error && <p className="form-error">{error}</p>}
    </div>
  );
}
