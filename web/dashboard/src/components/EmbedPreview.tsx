import { useEffect, useRef } from "react";

export function EmbedPreview({ videoId }: { videoId: number }) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const script = document.createElement("script");
    script.src = "/player.js";
    script.async = true;
    document.body.appendChild(script);
    return () => {
      document.body.removeChild(script);
    };
    // Re-run whenever the video changes so the new container gets initialized.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [videoId]);

  return <div ref={containerRef} data-clipfolio-video={videoId} className="embed-preview" />;
}
