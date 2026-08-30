import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const OUT = path.join(__dirname, "assets", "sample-clip.mp4");
const DURATION = 8;

// testsrc2 + a sine tone: a license-clean, generated-on-the-fly clip so the
// demo never depends on (or ships) third-party video.
execFileSync(
  "ffmpeg",
  [
    "-y",
    "-f", "lavfi", "-i", `testsrc2=size=640x360:rate=30:duration=${DURATION}`,
    "-f", "lavfi", "-i", `sine=frequency=440:duration=${DURATION}`,
    "-c:v", "libx264", "-pix_fmt", "yuv420p",
    "-c:a", "aac", "-shortest",
    OUT,
  ],
  { stdio: "inherit" },
);

console.log(`wrote ${OUT}`);
