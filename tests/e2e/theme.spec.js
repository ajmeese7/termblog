const { test, expect } = require("@playwright/test");

// Background hex per theme — must match background_hex in web/src/theme.rs.
// Values are stored as hex; tests compare against rgb() strings produced by
// getComputedStyle.
const THEME_BG_HEX = {
  pipboy: "#0a0a0a",
  dracula: "#282a36",
  nord: "#2e3440",
  monokai: "#272822",
  monochrome: "#000000",
  amber: "#0d0800",
  matrix: "#0d0208",
  paper: "#f5f5dc",
  terminal: "#1e1e1e",
};

const THEME_KEYS = [
  "pipboy", "dracula", "nord", "monokai", "monochrome",
  "amber", "matrix", "paper", "terminal",
];

function hexToRgb(hex) {
  const h = hex.replace("#", "");
  const r = parseInt(h.slice(0, 2), 16);
  const g = parseInt(h.slice(2, 4), 16);
  const b = parseInt(h.slice(4, 6), 16);
  return `rgb(${r}, ${g}, ${b})`;
}

// The WASM TUI mounts under <div id="grid">. We wait for it to populate
// before issuing keystrokes.
async function waitForApp(page) {
  await page.waitForSelector("#grid pre", { timeout: 15000 });
  await page.waitForTimeout(500);
}

async function getDocBg(page) {
  return page.evaluate(() =>
    window.getComputedStyle(document.documentElement).backgroundColor,
  );
}

async function getSavedTheme(page) {
  return page.evaluate(() => localStorage.getItem("termblog-theme"));
}

async function fetchDefaultTheme(page) {
  const res = await page.request.get("/api/config");
  const cfg = await res.json();
  return cfg.default_theme || cfg.defaultTheme;
}

async function openThemePicker(page) {
  await page.locator("body").click();
  await page.waitForTimeout(150);
  await page.keyboard.press("t");
  await page.waitForTimeout(500);
}

// Move the cursor in the theme picker from `from` to `to` (both theme keys),
// then press Enter to commit. Themes are listed in THEME_KEYS order; the
// cursor opens at the currently-selected theme.
async function selectThemeFrom(page, from, to) {
  const start = THEME_KEYS.indexOf(from);
  const end = THEME_KEYS.indexOf(to);
  if (start < 0 || end < 0) throw new Error(`unknown theme ${from} or ${to}`);

  await openThemePicker(page);

  const delta = end - start;
  const key = delta >= 0 ? "ArrowDown" : "ArrowUp";
  for (let i = 0; i < Math.abs(delta); i++) {
    await page.keyboard.press(key);
    await page.waitForTimeout(120);
  }
  await page.keyboard.press("Enter");
  await page.waitForTimeout(500);
}

test.describe("Web Theme Sync", () => {
  test("initial load uses server config theme", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const expected = await fetchDefaultTheme(page);
    expect(THEME_BG_HEX[expected]).toBeDefined();
    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[expected]));
  });

  test("theme change updates page background live", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    // Pick any other theme that isn't the start theme.
    const target = THEME_KEYS.find((k) => k !== start && k !== "terminal");
    await selectThemeFrom(page, start, target);

    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[target]));
  });

  test("theme change saves to localStorage", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    const target = THEME_KEYS.find((k) => k !== start && k !== "terminal");
    await selectThemeFrom(page, start, target);

    expect(await getSavedTheme(page)).toBe(target);
  });

  test("theme persists across page reload", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    const target = THEME_KEYS.find((k) => k !== start && k !== "terminal");
    await selectThemeFrom(page, start, target);
    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[target]));

    await page.reload({ waitUntil: "domcontentloaded" });
    await waitForApp(page);

    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[target]));
  });

  test("light theme (paper) applies correctly", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    await selectThemeFrom(page, start, "paper");

    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX.paper));
  });

  test("preview updates background while browsing in picker", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    const startIdx = THEME_KEYS.indexOf(start);
    const nextIdx = (startIdx + 1) % THEME_KEYS.length;
    const next = THEME_KEYS[nextIdx];

    await openThemePicker(page);
    const direction = nextIdx > startIdx ? "ArrowDown" : "ArrowUp";
    await page.keyboard.press(direction);
    await page.waitForTimeout(400);

    if (next !== "terminal") {
      expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[next]));
    }
  });

  test("escape reverts background to original theme", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const start = await fetchDefaultTheme(page);
    await openThemePicker(page);
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(400);
    await page.keyboard.press("Escape");
    await page.waitForTimeout(400);

    expect(await getDocBg(page)).toBe(hexToRgb(THEME_BG_HEX[start]));
  });
});
