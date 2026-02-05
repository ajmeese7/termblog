const { test, expect } = require("@playwright/test");

// Expected theme background colors (must match termBlogThemes in index.html)
const THEME_BACKGROUNDS = {
  pipboy: "rgb(10, 10, 10)", // #0a0a0a
  dracula: "rgb(40, 42, 54)", // #282a36
  nord: "rgb(46, 52, 64)", // #2e3440
  monokai: "rgb(39, 40, 34)", // #272822
  monochrome: "rgb(0, 0, 0)", // #000000
  amber: "rgb(13, 8, 0)", // #0d0800
  matrix: "rgb(13, 2, 8)", // #0d0208
  paper: "rgb(245, 245, 220)", // #f5f5dc
};

// Helper: get the current page background color
async function getPageBackground(page) {
  return page.evaluate(() => {
    return window.getComputedStyle(document.documentElement).backgroundColor;
  });
}

// Helper: get localStorage theme value
async function getSavedTheme(page) {
  return page.evaluate(() => {
    return localStorage.getItem("termblog-theme");
  });
}

// Helper: wait for terminal WebSocket to connect
async function waitForTerminal(page) {
  await page.waitForTimeout(3000);
}

// Helper: open theme selector
async function openThemeSelector(page) {
  await page.click("#terminal");
  await page.waitForTimeout(500);
  await page.keyboard.type("t");
  await page.waitForTimeout(1500);
}

// Helper: open theme selector and pick a theme by navigating from current position
async function selectTheme(page, stepsDown) {
  await openThemeSelector(page);

  for (let i = 0; i < stepsDown; i++) {
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(300);
  }

  await page.keyboard.press("Enter");
  await page.waitForTimeout(2000);
}

test.describe("Web Theme Sync", () => {
  test("initial load uses server config theme", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Config default is "dracula" - page should match
    const bg = await getPageBackground(page);
    expect(bg).toBe(THEME_BACKGROUNDS.dracula);
  });

  test("theme change updates page background via OSC sequence", async ({
    page,
  }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Theme selector starts at dracula (config default, index 1)
    // Move 1 down to nord (index 2)
    await selectTheme(page, 1);

    const bg = await getPageBackground(page);
    expect(bg).toBe(THEME_BACKGROUNDS.nord);
  });

  test("theme change saves to localStorage", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Select nord
    await selectTheme(page, 1);

    const saved = await getSavedTheme(page);
    expect(saved).toBe("nord");
  });

  test("theme persists across page reload", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Select nord
    await selectTheme(page, 1);
    expect(await getPageBackground(page)).toBe(THEME_BACKGROUNDS.nord);

    // Reload - should restore from localStorage
    await page.reload({ waitUntil: "domcontentloaded" });
    await page.waitForTimeout(3000);

    // Page background should persist from localStorage
    const bg = await getPageBackground(page);
    expect(bg).toBe(THEME_BACKGROUNDS.nord);

    // xterm viewport should also use the saved theme (TUI started with it)
    const xtermBg = await page.evaluate(() => {
      const viewport = document.querySelector(".xterm-viewport");
      return viewport
        ? window.getComputedStyle(viewport).backgroundColor
        : null;
    });
    expect(xtermBg).toBe(THEME_BACKGROUNDS.nord);
  });

  test("light theme (paper) applies correctly", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // From dracula (index 1), paper is 6 steps down (index 7)
    await selectTheme(page, 6);

    const bg = await getPageBackground(page);
    expect(bg).toBe(THEME_BACKGROUNDS.paper);
  });

  test("theme preview updates page background while browsing", async ({
    page,
  }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    await openThemeSelector(page);

    // Cursor starts at dracula - move down to nord
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(1000);
    expect(await getPageBackground(page)).toBe(THEME_BACKGROUNDS.nord);

    // Move down to monokai
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(1000);
    expect(await getPageBackground(page)).toBe(THEME_BACKGROUNDS.monokai);
  });

  test("cancel reverts page background to original theme", async ({
    page,
  }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Preview a different theme
    await openThemeSelector(page);
    await page.keyboard.press("ArrowDown"); // nord
    await page.waitForTimeout(1000);
    expect(await getPageBackground(page)).toBe(THEME_BACKGROUNDS.nord);

    // Cancel - should revert to dracula
    await page.keyboard.press("Escape");
    await page.waitForTimeout(1500);
    expect(await getPageBackground(page)).toBe(THEME_BACKGROUNDS.dracula);

    // localStorage should be reverted to the original theme, not the previewed one
    const saved = await getSavedTheme(page);
    expect(saved).not.toBe("nord");
  });

  test("xterm terminal theme updates with page", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // Select nord
    await selectTheme(page, 1);

    // Check that the xterm viewport background also changed
    const xtermBg = await page.evaluate(() => {
      const viewport = document.querySelector(".xterm-viewport");
      return viewport
        ? window.getComputedStyle(viewport).backgroundColor
        : null;
    });

    // xterm viewport should have nord background
    expect(xtermBg).toBe(THEME_BACKGROUNDS.nord);
  });
});

test.describe("Web Terminal Connection", () => {
  test("terminal connects and displays content", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    // The loading indicator should be hidden
    const loadingVisible = await page.evaluate(() => {
      const el = document.getElementById("loading");
      return el && !el.classList.contains("hidden");
    });
    expect(loadingVisible).toBe(false);

    // Terminal element should exist
    const termExists = await page.evaluate(() => {
      return !!document.querySelector(".xterm");
    });
    expect(termExists).toBe(true);
  });

  test("terminal container background matches theme", async ({ page }) => {
    await page.goto("/", { waitUntil: "networkidle" });
    await waitForTerminal(page);

    const containerBg = await page.evaluate(() => {
      const container = document.getElementById("terminal-container");
      return container
        ? window.getComputedStyle(container).backgroundColor
        : null;
    });

    // Should match the default theme (dracula)
    expect(containerBg).toBe(THEME_BACKGROUNDS.dracula);
  });
});
