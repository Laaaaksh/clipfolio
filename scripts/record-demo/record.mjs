import { chromium } from "@playwright/test";
import { fileURLToPath } from "node:url";
import fs from "node:fs";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const BASE_URL = process.env.CLIPFOLIO_BASE_URL || "http://localhost:8080";
const CLIP_PATH = path.join(__dirname, "assets", "sample-clip.mp4");
const OUT_DIR = path.join(__dirname, "assets", "video");
const HEADLESS = process.env.HEADLESS !== "false";

const ADMIN_EMAIL = "demo@clipfolio.local";
const ADMIN_PASSWORD = "DemoPass123!";
const VIDEO_TITLE = "Product Launch Teaser";
const CTA_LABEL = "Book a demo";
const CTA_URL = "https://example.com/book-a-demo";
const GATE_HEADLINE = "Get the download link";
const VIEWER_NAME = "Ada Lovelace";
const VIEWER_EMAIL = "ada@example.com";

if (!fs.existsSync(CLIP_PATH)) {
  console.error(`missing ${CLIP_PATH} - run \`npm run clip\` first`);
  process.exit(1);
}

fs.rmSync(OUT_DIR, { recursive: true, force: true });
fs.mkdirSync(OUT_DIR, { recursive: true });

const browser = await chromium.launch({ headless: HEADLESS });
const context = await browser.newContext({
  viewport: { width: 1280, height: 800 },
  deviceScaleFactor: 2,
  recordVideo: { dir: OUT_DIR, size: { width: 1280, height: 800 } },
});
const page = await context.newPage();
const mainVideo = page.video();

// Auto-close the tab the CTA opens (real target="_blank" navigation) so the
// recording never leaves the dashboard/preview page it's actually capturing.
// Playwright records every page in the context, so the popup gets its own
// (throwaway) video file - we only keep mainVideo's below.
context.on("page", (popup) => {
  if (popup !== page) popup.close().catch(() => {});
});

try {
  await page.goto(BASE_URL);
  await page.waitForTimeout(1200);

  await page.getByLabel("Email").fill(ADMIN_EMAIL);
  await page.getByLabel("Password").fill(ADMIN_PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await page.getByRole("heading", { name: "Videos" }).waitFor();
  await page.waitForTimeout(2500);

  await page.getByPlaceholder("New video title").fill(VIDEO_TITLE);
  await page.getByRole("button", { name: "Create" }).click();
  await page.getByRole("heading", { name: VIDEO_TITLE }).waitFor();

  await page.locator('input[type="file"]').setInputFiles(CLIP_PATH);
  await page.getByRole("heading", { name: "Preview" }).waitFor({ timeout: 30000 });
  await page.waitForTimeout(2500);

  await page.getByPlaceholder("Button label").fill(CTA_LABEL);
  await page.getByPlaceholder("https://…").fill(CTA_URL);
  await page.getByRole("button", { name: "Add" }).click();
  await page.getByText(CTA_LABEL, { exact: true }).waitFor();
  await page.waitForTimeout(2800);

  const leadGatePanel = page.locator(".panel", { has: page.getByRole("heading", { name: "Lead-capture gate" }) });
  await leadGatePanel.getByRole("checkbox", { name: /Require an email/ }).check();
  await leadGatePanel.getByLabel("Headline").fill(GATE_HEADLINE);
  await leadGatePanel.getByRole("checkbox", { name: "Also require a name" }).check();
  await leadGatePanel.getByRole("button", { name: "Save" }).click();
  await leadGatePanel.getByText("Saved", { exact: true }).waitFor();
  await page.waitForTimeout(2800);

  const preview = page.locator(".embed-preview");
  await preview.scrollIntoViewIfNeeded();
  const gate = preview.locator(".clipfolio-player__gate");
  await gate.waitFor({ timeout: 15000 });
  await page.waitForTimeout(1500);

  await gate.locator('input[name="name"]').fill(VIEWER_NAME);
  await gate.locator('input[name="email"]').fill(VIEWER_EMAIL);
  await page.waitForTimeout(1000);
  await gate.getByRole("button", { name: "Continue watching" }).click();
  await gate.waitFor({ state: "detached", timeout: 5000 });

  const ctaButton = preview.locator(".clipfolio-player__cta");
  await ctaButton.waitFor({ timeout: 20000 });
  await page.waitForTimeout(2000);
  await ctaButton.click();
  await page.waitForTimeout(2000);

  await page.getByRole("heading", { name: "Analytics" }).scrollIntoViewIfNeeded();
  await page.waitForTimeout(8000);
  await page.getByRole("heading", { name: "Leads" }).scrollIntoViewIfNeeded();
  await page.waitForTimeout(7000);
} finally {
  await context.close();
  await browser.close();
}

const savedPath = await mainVideo.path();
const finalPath = path.join(OUT_DIR, "raw.webm");
fs.renameSync(savedPath, finalPath);

// Discard the popup tab's own (throwaway) recording - only the main page matters.
for (const f of fs.readdirSync(OUT_DIR)) {
  if (f !== "raw.webm") fs.rmSync(path.join(OUT_DIR, f));
}

console.log(`recorded ${finalPath}`);
