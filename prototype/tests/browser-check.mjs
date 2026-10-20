import { chromium } from "playwright-core";
import fs from "node:fs";
import path from "node:path";
import assert from "node:assert/strict";

const CHANNEL = process.env.PW_CHANNEL || "chrome";
const BASE_URL = process.env.BASE_URL || "http://127.0.0.1:4173/";
const OUT = path.join(import.meta.dirname, "shots");
fs.mkdirSync(OUT, { recursive: true });

const norm = (value) => value.replace(/\s+/g, " ").trim();
const errors = [];
const pageErrors = [];

const skipLinkFocus = async (page) => {
  await page.keyboard.press("Tab");
  return page.evaluate(() => document.activeElement?.className || "");
};

const browser = await chromium.launch({ channel: CHANNEL, headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
const page = await context.newPage();

page.on("console", (msg) => {
  if (msg.type() === "error") errors.push(msg.text());
});
page.on("pageerror", (err) => pageErrors.push(err.message));

await page.goto(BASE_URL, { waitUntil: "networkidle" });
await page.waitForSelector('#main[aria-busy="false"]');
await page.waitForFunction(() => document.querySelector("#ledger-list").children.length === 3);
await page.evaluate(() => document.fonts.ready);

assert.match(await skipLinkFocus(page), /skip-link/);
await page.keyboard.press("Enter");
assert.equal(new URL(page.url()).hash, "#main");

assert.equal(norm(await page.locator("#price").innerText()), "Rp 7.200.000");
assert.equal(norm(await page.locator("#live-label").innerText()), "LIVE");
assert.equal(norm(await page.locator("#bid-count").innerText()), "3 bid");
assert.equal(await page.locator("#amount-input").inputValue(), "7.300.000");
await page.screenshot({ path: path.join(OUT, "desktop-initial.png"), fullPage: true, animations: "disabled" });

// Name, validation, and a successful bid.
await page.locator("#name-input").fill("Raihan");
await page.locator("#name-input").press("Enter");
await page.waitForSelector("#identity-summary:not([hidden])");
assert.match(norm(await page.locator("#identity-summary").innerText()), /Bid sebagai Raihan/);
assert.equal(await page.locator("#bid-submit").isEnabled(), true);

await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector('#main[aria-busy="false"]');
await page.waitForSelector("#identity-summary:not([hidden])");
assert.match(norm(await page.locator("#identity-summary").innerText()), /Bid sebagai Raihan/);

await page.locator("#amount-input").fill("7000000");
await page.locator("#bid-submit").click();
await page.waitForSelector("#bid-error:not([hidden])");
assert.match(norm(await page.locator("#bid-error").innerText()), /Bid minimum Rp 7\.300\.000/);
assert.equal(await page.locator("#amount-input").inputValue(), "7.300.000");

await page.locator("#bid-submit").click();
await page.waitForFunction(() => document.querySelector("#price").textContent.includes("7.300.000"));
assert.equal(norm(await page.locator("#price").innerText()), "Rp 7.300.000");
assert.match(norm(await page.locator("#ledger-list li:first-child").innerText()), /Raihan/);

// A second tab in the same context shares localStorage and BroadcastChannel.
const page2 = await context.newPage();
page2.on("console", (msg) => {
  if (msg.type() === "error") errors.push(`tab2: ${msg.text()}`);
});
page2.on("pageerror", (err) => pageErrors.push(`tab2: ${err.message}`));
await page2.goto(BASE_URL, { waitUntil: "networkidle" });
await page2.waitForSelector('#main[aria-busy="false"]');
await page2.waitForFunction(() => document.querySelector("#price").textContent.includes("7.300.000"));
await page2.locator("#name-input").fill("Dewi");
await page2.locator('#name-form button[type="submit"]').click();
await page2.waitForSelector("#identity-summary:not([hidden])");

await page.waitForFunction(() => document.querySelector("#price-meta").textContent.includes("2 penonton"), null, {
  timeout: 7000,
});

await page2.locator("#amount-input").fill("7400000");
await page2.locator("#bid-submit").click();
await page2.waitForFunction(() => document.querySelector("#price").textContent.includes("7.400.000"));

await page.waitForSelector("#toast:not([hidden])", { timeout: 7000 });
const outbidMessage = norm(await page.locator("#toast-message").innerText());
assert.match(outbidMessage, /Kamu di-outbid/);
assert.match(outbidMessage, /Rp 7\.400\.000/);
assert.match(outbidMessage, /Dewi/);
assert.equal(norm(await page.locator("#price").innerText()), "Rp 7.400.000");
assert.equal(await page.locator("#bid-status").isVisible(), false);
await page.screenshot({ path: path.join(OUT, "desktop-outbid.png"), animations: "disabled" });

await page.locator("#toast-close").click();
assert.equal(await page.locator("#toast").isVisible(), false);

await page.locator("#change-name").click();
await page.waitForSelector("#name-form:not([hidden])");
assert.equal(await page.locator("#name-input").inputValue(), "Raihan");
await page.locator("#name-input").press("Enter");
await page.waitForSelector("#identity-summary:not([hidden])");

await page2.close();
await page.waitForFunction(() => document.querySelector("#price-meta").textContent.includes("1 penonton"), null, {
  timeout: 9000,
});

// Reset, then exercise the built-in rival simulation.
page.once("dialog", (dialog) => dialog.accept());
await page.locator("#reset-demo").click();
await page.waitForFunction(() => document.querySelector("#price").textContent.includes("6.500.000"));
assert.equal(norm(await page.locator("#bid-count").innerText()), "0 bid");
assert.equal(await page.locator("#ledger-empty").isVisible(), true);

await page.locator("#quick-bids button").nth(1).click();
assert.equal(await page.locator("#amount-input").inputValue(), "6.600.000");
await page.locator("#quick-bids button").nth(0).click();
assert.equal(await page.locator("#amount-input").inputValue(), "6.500.000");

await page.locator("#bid-submit").click();
await page.waitForFunction(() => document.querySelector("#price").textContent.includes("6.500.000"));
await page.locator("#simulate-rival").click();
await page.waitForFunction(() => document.querySelector("#price").textContent.includes("6.600.000"));
await page.waitForSelector("#toast:not([hidden])");
assert.match(norm(await page.locator("#toast-message").innerText()), /di-outbid/);

// Focus and target-size checks.
const focusOutline = await page.evaluate(() => {
  const input = document.querySelector("#amount-input");
  input.focus();
  return getComputedStyle(document.querySelector(".amount-field")).outlineWidth;
});
assert.equal(focusOutline, "3px");

const smallTargets = await page.evaluate(() =>
  [...document.querySelectorAll("button, input")]
    .filter((el) => {
      const rect = el.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.height < 44;
    })
    .map((el) => `${el.tagName}:${el.textContent.trim() || el.id}:${el.getBoundingClientRect().height}`),
);
assert.deepEqual(smallTargets, []);

const unnamedButtons = await page.evaluate(() =>
  [...document.querySelectorAll("button")]
    .filter((el) => !(el.textContent.trim() || el.getAttribute("aria-label")))
    .map((el) => el.outerHTML),
);
assert.deepEqual(unnamedButtons, []);

const unlabeledInputs = await page.evaluate(() =>
  [...document.querySelectorAll("input")]
    .filter((el) => !(el.labels?.length || el.getAttribute("aria-label")))
    .map((el) => el.outerHTML),
);
assert.deepEqual(unlabeledInputs, []);

// Responsive checks at tablet, phone, and narrow phone widths.
const sizes = [
  { width: 768, height: 1024, name: "tablet" },
  { width: 390, height: 844, name: "mobile" },
  { width: 320, height: 740, name: "mobile-narrow" },
];
for (const size of sizes) {
  await page.setViewportSize({ width: size.width, height: size.height });
  await page.reload({ waitUntil: "networkidle" });
  await page.waitForSelector('#main[aria-busy="false"]');
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth + 1,
  );
  assert.equal(overflow, false, `horizontal overflow at ${size.width}px`);
  await page.screenshot({
    path: path.join(OUT, `${size.name}.png`),
    fullPage: true,
    animations: "disabled",
  });
}

// Ended with a winner.
await page.setViewportSize({ width: 1440, height: 1100 });
await page.evaluate(() => {
  const key = "lelang-demo-state-v2";
  const state = JSON.parse(localStorage.getItem(key));
  const live = state.auctions.find((auction) => auction.status === "live");
  live.endsAt = new Date(Date.now() - 1000).toISOString();
  localStorage.setItem(key, JSON.stringify(state));
});
await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector("#result:not([hidden])");
assert.match(norm(await page.locator("#result-title").innerText()), /Penantang demo memenangkan lot ini/);
assert.match(norm(await page.locator("#result-price").innerText()), /Rp 6\.600\.000/);
await page.screenshot({ path: path.join(OUT, "desktop-ended-winner.png"), fullPage: true, animations: "disabled" });

// Ended without bids.
page.once("dialog", (dialog) => dialog.accept());
await page.locator("#reset-result").click();
await page.waitForFunction(() => document.querySelector("#price").textContent.includes("6.500.000"));
await page.evaluate(() => {
  const key = "lelang-demo-state-v2";
  const state = JSON.parse(localStorage.getItem(key));
  const live = state.auctions.find((auction) => auction.status === "live");
  live.endsAt = new Date(Date.now() - 1000).toISOString();
  localStorage.setItem(key, JSON.stringify(state));
});
await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector("#result:not([hidden])");
assert.match(norm(await page.locator("#result-title").innerText()), /Tidak ada bid/);
assert.match(norm(await page.locator("#result-price").innerText()), /Tidak ada harga akhir/);
assert.match(norm(await page.locator("#ledger-empty").innerText()), /Tidak ada bid pada lelang ini/);
await page.screenshot({ path: path.join(OUT, "desktop-ended-unsold.png"), fullPage: true, animations: "disabled" });

// Corrupt state recovery.
await page.evaluate(() => localStorage.setItem("lelang-demo-state-v2", "{broken"));
await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector("#app-banner:not([hidden])");
assert.match(norm(await page.locator("#app-banner").innerText()), /tidak terbaca/);
await page.screenshot({ path: path.join(OUT, "desktop-error.png"), fullPage: true, animations: "disabled" });

assert.deepEqual(errors, [], `console errors: ${errors.join(" | ")}`);
assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join(" | ")}`);

await browser.close();
console.log(
  JSON.stringify(
    {
      status: "PASS",
      checks: [
        "initial data and 3 seeded bids",
        "skip link receives focus and jumps to main",
        "name persists per tab across reload",
        "keyboard Enter submits the name form",
        "below-minimum bid rejected",
        "successful bid updates price and ledger",
        "second tab sees price and presence",
        "outbid toast on first tab and stale status cleared",
        "toast close works",
        "change name control works",
        "presence drops after tab closes",
        "reset clears bids",
        "quick bid buttons update the amount",
        "rival simulation triggers outbid",
        "ended winner state",
        "ended unsold state",
        "corrupt state recovery banner",
        "3px focus outline",
        "all targets >= 44px",
        "all buttons named and inputs labelled",
        "no horizontal overflow at 768/390/320",
        "zero console and page errors",
      ],
      screenshots: OUT,
    },
    null,
    2,
  ),
);
