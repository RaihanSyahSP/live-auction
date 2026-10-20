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

const browser = await chromium.launch({ channel: CHANNEL, headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
const page = await context.newPage();

page.on("console", (msg) => {
  if (msg.type() === "error") errors.push(`admin: ${msg.text()}`);
});
page.on("pageerror", (err) => pageErrors.push(`admin: ${err.message}`));

await page.goto(`${BASE_URL}admin.html`, { waitUntil: "networkidle" });
await page.waitForSelector('#admin-main[aria-busy="false"]');
await page.waitForFunction(() => document.querySelectorAll(".admin-card").length >= 2);
await page.evaluate(() => document.fonts.ready);

assert.match(norm(await page.locator("#admin-summary").innerText()), /1 live · 1 draft/);
const liveCard = page.locator('[data-auction-id="1"]');
const draftCard = page.locator('[data-auction-id="2"]');
assert.match(norm(await liveCard.locator(".admin-status").innerText()), /LIVE/);
assert.match(norm(await draftCard.locator(".admin-status").innerText()), /DRAFT/);
assert.equal(await draftCard.locator('button[data-action="start"]').isDisabled(), true);
await page.screenshot({ path: path.join(OUT, "admin-initial.png"), fullPage: true, animations: "disabled" });

// End the seeded live auction, then start the seeded draft.
page.once("dialog", (dialog) => dialog.accept());
await liveCard.locator('button[data-action="end"]').click();
await page.waitForFunction(() => {
  const card = document.querySelector('[data-auction-id="1"]');
  return card?.querySelector(".admin-status")?.textContent.trim() === "SELESAI";
});
assert.match(norm(await liveCard.locator(".admin-meta").innerText()), /Peserta 03/);
assert.equal(await draftCard.locator('button[data-action="start"]').isEnabled(), true);

page.once("dialog", (dialog) => dialog.accept());
await draftCard.locator('button[data-action="start"]').click();
await page.waitForFunction(() => {
  const card = document.querySelector('[data-auction-id="2"]');
  return card?.querySelector(".admin-status")?.textContent.trim() === "LIVE";
});
assert.match(norm(await page.locator("#admin-summary").innerText()), /1 live/);
assert.equal(
  await page.locator('[data-auction-id="2"] a[href="index.html?auction=2"]').count(),
  1,
);

// Open the started auction in the room and place a bid.
const room = await context.newPage();
room.on("console", (msg) => {
  if (msg.type() === "error") errors.push(`room: ${msg.text()}`);
});
room.on("pageerror", (err) => pageErrors.push(`room: ${err.message}`));
await room.goto(`${BASE_URL}index.html?auction=2`, { waitUntil: "networkidle" });
await room.waitForSelector('#main[aria-busy="false"]');
assert.equal(norm(await room.locator("#live-label").innerText()), "LIVE");
assert.equal(norm(await room.locator("#price").innerText()), "Rp 8.500.000");
assert.equal(norm(await room.locator("#bid-count").innerText()), "0 bid");

await room.locator("#name-input").fill("Admin Test Bidder");
await room.locator("#name-input").press("Enter");
await room.waitForSelector("#identity-summary:not([hidden])");
await room.locator("#bid-submit").click();
await room.waitForFunction(() => document.querySelector("#price").textContent.includes("8.500.000"));
assert.match(norm(await room.locator("#ledger-list li:first-child").innerText()), /Admin Test Bidder/);

// Admin list receives the bid through BroadcastChannel.
await page.waitForFunction(() => {
  const card = document.querySelector('[data-auction-id="2"]');
  return card?.textContent.includes("8.500.000") && card?.textContent.includes("1 bid");
});

// End from admin, and the room must show the winner.
page.once("dialog", (dialog) => dialog.accept());
await page.locator('[data-auction-id="2"] button[data-action="end"]').click();
await page.waitForFunction(() => {
  const card = document.querySelector('[data-auction-id="2"]');
  return card?.querySelector(".admin-status")?.textContent.trim() === "SELESAI";
});
await room.waitForSelector("#result:not([hidden])");
assert.match(norm(await room.locator("#result-title").innerText()), /Admin Test Bidder memenangkan lot ini/);
assert.match(norm(await room.locator("#result-price").innerText()), /Rp 8\.500\.000/);
await page.screenshot({ path: path.join(OUT, "admin-ended.png"), fullPage: true, animations: "disabled" });

// Create a new draft and check validation and deletion.
await page.locator("#auction-title").fill("Kamera uji admin");
await page.locator("#auction-subtitle").fill("Body only");
await page.locator("#auction-description").fill("Kamera untuk menguji alur draft dari halaman admin.");
await page.locator("#auction-condition").fill("Used, 8/10");
await page.locator("#auction-includes").fill("Body, baterai");
await page.locator("#starting-price").fill("1000000");
await page.locator("#min-increment").fill("50000");
await page.locator("#duration").fill("5");
await page.locator("#create-submit").click();
await page.waitForSelector("#form-status:not([hidden])");
await page.waitForFunction(() =>
  [...document.querySelectorAll(".admin-card-title")].some((element) =>
    element.textContent.includes("Kamera uji admin"),
  ),
);

const newCard = page.locator(".admin-card", { hasText: "Kamera uji admin" });
assert.match(norm(await newCard.locator(".admin-status").innerText()), /DRAFT/);
assert.match(norm(await newCard.locator(".admin-meta").innerText()), /Rp 1\.000\.000/);
assert.equal(await page.locator("#starting-price").inputValue(), "");
await page.screenshot({ path: path.join(OUT, "admin-draft-created.png"), fullPage: true, animations: "disabled" });

const newId = await newCard.getAttribute("data-auction-id");
page.once("dialog", (dialog) => dialog.accept());
await newCard.locator('button[data-action="delete"]').click();
await page.waitForFunction(
  (id) => !document.querySelector(`[data-auction-id="${id}"]`),
  newId,
);
assert.equal(await page.locator(`[data-auction-id="${newId}"]`).count(), 0);

await page.locator("#create-submit").click();
await page.waitForSelector("#form-error:not([hidden])");
assert.match(norm(await page.locator("#form-error").innerText()), /Judul minimal 3 karakter/);

// Restore the showcase state.
page.once("dialog", (dialog) => dialog.accept());
await page.locator("#reset-all").click();
await page.waitForFunction(() => {
  const cards = [...document.querySelectorAll(".admin-card")];
  return (
    cards.some((card) => card.querySelector(".admin-status")?.textContent.trim() === "LIVE") &&
    cards.some((card) => card.querySelector(".admin-status")?.textContent.trim() === "DRAFT")
  );
});

// Responsive and target checks.
const sizes = [
  { width: 768, height: 1024, name: "admin-tablet" },
  { width: 390, height: 844, name: "admin-mobile" },
  { width: 320, height: 740, name: "admin-mobile-narrow" },
];
for (const size of sizes) {
  await page.setViewportSize({ width: size.width, height: size.height });
  await page.reload({ waitUntil: "networkidle" });
  await page.waitForSelector('#admin-main[aria-busy="false"]');
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth + 1,
  );
  assert.equal(overflow, false, `admin horizontal overflow at ${size.width}px`);
  await page.screenshot({
    path: path.join(OUT, `${size.name}.png`),
    fullPage: true,
    animations: "disabled",
  });
}

const smallTargets = await page.evaluate(() =>
  [...document.querySelectorAll("button, input, textarea")]
    .filter((element) => {
      const rect = element.getBoundingClientRect();
      return rect.width > 0 && rect.height > 0 && rect.height < 44;
    })
    .map((element) => `${element.tagName}:${element.textContent.trim() || element.id}`),
);
assert.deepEqual(smallTargets, []);

const unnamedButtons = await page.evaluate(() =>
  [...document.querySelectorAll("button")]
    .filter((element) => !(element.textContent.trim() || element.getAttribute("aria-label")))
    .map((element) => element.outerHTML),
);
assert.deepEqual(unnamedButtons, []);

const unlabeledInputs = await page.evaluate(() =>
  [...document.querySelectorAll("input, textarea")]
    .filter((element) => !(element.labels?.length || element.getAttribute("aria-label")))
    .map((element) => element.outerHTML),
);
assert.deepEqual(unlabeledInputs, []);

assert.deepEqual(errors, [], `console errors: ${errors.join(" | ")}`);
assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join(" | ")}`);

await browser.close();
console.log(
  JSON.stringify(
    {
      status: "PASS",
      checks: [
        "admin loads with live and draft auction",
        "start is blocked while another auction is live",
        "end live auction moves it to ended",
        "start draft moves it to live",
        "room opens the started auction",
        "bid in room updates admin price and bid count",
        "end from admin updates room winner and final price",
        "create draft through form",
        "delete draft",
        "empty form validation",
        "reset all restores live and draft",
        "no horizontal overflow at 768/390/320",
        "all targets >= 44px",
        "all buttons named and inputs labelled",
        "zero console and page errors",
      ],
      screenshots: OUT,
    },
    null,
    2,
  ),
);
