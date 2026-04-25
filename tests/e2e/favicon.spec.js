const { test, expect } = require("@playwright/test");

async function waitForApp(page) {
  await page.waitForSelector("#grid pre", { timeout: 15000 });
  await page.waitForTimeout(500);
}

async function getFaviconHref(page) {
  return page.evaluate(() => {
    const link = document.querySelector('link[rel="icon"]');
    return link ? link.getAttribute("href") : null;
  });
}

test.describe("Theme-aware Favicon", () => {
  test("server injects favicon link and mode meta", async ({ page }) => {
    await page.goto("/", { waitUntil: "domcontentloaded" });

    const href = await getFaviconHref(page);
    expect(href).toBeTruthy();

    const mode = await page.evaluate(() => {
      const m = document.querySelector('meta[name="termblog-favicon-mode"]');
      return m ? m.getAttribute("content") : null;
    });
    expect(["letter", "emoji", "image"]).toContain(mode);
  });

  test("/favicon serves an SVG with the expected content type", async ({ page }) => {
    const res = await page.request.get("/favicon");
    expect(res.status()).toBe(200);

    const ctype = res.headers()["content-type"] || "";
    // letter and emoji modes return svg; image mode might return any image
    // type — both are acceptable as long as it's an image.
    expect(ctype).toMatch(/image\//);
  });

  test("/favicon honours ?theme= query for letter/emoji modes", async ({ page }) => {
    const modeRes = await page.request.get("/favicon");
    const ctype = modeRes.headers()["content-type"] || "";
    test.skip(!ctype.includes("svg+xml"), "non-SVG mode (image); theme query is not relevant");

    const draculaRes = await page.request.get("/favicon?theme=dracula");
    const monoRes = await page.request.get("/favicon?theme=monochrome");
    const dracBody = await draculaRes.text();
    const monoBody = await monoRes.text();

    expect(dracBody).not.toBe(monoBody);
    // Dracula background hex appears in its SVG; monochrome's does not.
    expect(dracBody.toLowerCase()).toContain("#282a36");
    expect(monoBody.toLowerCase()).toContain("#000000");
  });

  test("favicon link href updates when the user switches theme", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForApp(page);

    const before = await getFaviconHref(page);

    // Open theme picker and commit a new selection
    await page.locator("body").click();
    await page.waitForTimeout(150);
    await page.keyboard.press("t");
    await page.waitForTimeout(500);
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(200);
    await page.keyboard.press("Enter");
    await page.waitForTimeout(500);

    const after = await getFaviconHref(page);
    expect(after).not.toBe(before);
    expect(after).toMatch(/\/favicon\?theme=/);
  });
});
