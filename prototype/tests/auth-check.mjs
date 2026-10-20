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

async function checkBasics(page, label) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth + 1,
  );
  assert.equal(overflow, false, `${label}: horizontal overflow`);

  const smallTargets = await page.evaluate(() =>
    [...document.querySelectorAll("button, input, textarea")]
      .filter((element) => {
        const rect = element.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0 && rect.height < 44;
      })
      .map((element) => `${element.tagName}:${element.textContent.trim() || element.id}`),
  );
  assert.deepEqual(smallTargets, [], `${label}: small targets`);

  const unnamedButtons = await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .filter((element) => !(element.textContent.trim() || element.getAttribute("aria-label")))
      .map((element) => element.outerHTML),
  );
  assert.deepEqual(unnamedButtons, [], `${label}: unnamed buttons`);

  const unlabeledInputs = await page.evaluate(() =>
    [...document.querySelectorAll("input, textarea")]
      .filter((element) => !(element.labels?.length || element.getAttribute("aria-label")))
      .map((element) => element.outerHTML),
  );
  assert.deepEqual(unlabeledInputs, [], `${label}: unlabeled inputs`);
}

const browser = await chromium.launch({ channel: CHANNEL, headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1100 } });
const page = await context.newPage();

page.on("console", (msg) => {
  if (msg.type() === "error") errors.push(msg.text());
});
page.on("pageerror", (err) => pageErrors.push(err.message));

// Guest can watch but cannot bid.
await page.goto(`${BASE_URL}index.html`, { waitUntil: "networkidle" });
await page.waitForSelector('#main[aria-busy="false"]');
assert.equal(await page.locator("#identity-guest").isVisible(), true);
assert.equal(await page.locator("#bid-form").isVisible(), false);
assert.match(norm(await page.locator("#auth-links").innerText()), /Masuk/);
assert.match(await page.locator("#login-link").getAttribute("href"), /login\.html\?next=index\.html/);
await page.screenshot({ path: path.join(OUT, "guest-room.png"), fullPage: true, animations: "disabled" });

await page.setViewportSize({ width: 320, height: 740 });
await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector('#main[aria-busy="false"]');
await checkBasics(page, "guest room 320");
await page.screenshot({ path: path.join(OUT, "guest-room-320.png"), fullPage: true, animations: "disabled" });
await page.setViewportSize({ width: 1440, height: 1100 });

// Login validation and successful form login.
await page.goto(`${BASE_URL}login.html?next=index.html`, { waitUntil: "networkidle" });
await page.waitForSelector("#login-form");
await page.screenshot({ path: path.join(OUT, "login-page.png"), fullPage: true, animations: "disabled" });
await page.locator("#login-submit").click();
await page.waitForSelector("#login-error:not([hidden])");
assert.match(norm(await page.locator("#login-error").innerText()), /Isi email dan password/);

await page.locator("#email").fill("raihan@example.com");
await page.locator("#password").fill("salah");
await page.locator("#login-submit").click();
await page.waitForFunction(() => {
  const error = document.querySelector("#login-error");
  return error && !error.hidden && error.textContent.includes("Password salah");
});

await page.locator("#password").fill("demo123");
await page.locator("#password").press("Enter");
await page.waitForURL(/index\.html/);
await page.waitForSelector('#main[aria-busy="false"]');
assert.match(norm(await page.locator("#identity-summary").innerText()), /Bid sebagai Raihan/);
await page.screenshot({ path: path.join(OUT, "login-success.png"), fullPage: true, animations: "disabled" });

// Header logout returns the tab to guest mode.
await page.locator("#auth-links button").click();
await page.waitForURL(/index\.html/);
await page.waitForSelector('#main[aria-busy="false"]');
assert.equal(await page.locator("#identity-guest").isVisible(), true);
assert.equal(await page.locator("#bid-form").isVisible(), false);

// Register with validation, then land on the account page.
await page.goto(`${BASE_URL}register.html?next=account.html`, { waitUntil: "networkidle" });
await page.waitForSelector("#register-form");
await page.screenshot({ path: path.join(OUT, "register-page.png"), fullPage: true, animations: "disabled" });
await page.locator("#register-submit").click();
await page.waitForSelector("#register-error:not([hidden])");
assert.match(norm(await page.locator("#register-error").innerText()), /Nama minimal 2 karakter/);

await page.locator("#name").fill("Penguji Baru");
await page.locator("#email").fill("penguji@example.com");
await page.locator("#password").fill("rahasia123");
await page.locator("#confirm-password").fill("beda123");
await page.locator("#register-submit").click();
await page.waitForSelector("#register-error:not([hidden])");
assert.match(norm(await page.locator("#register-error").innerText()), /belum sama/);

await page.locator("#confirm-password").fill("rahasia123");
await page.locator("#register-submit").click();
await page.waitForURL(/account\.html/);
await page.waitForSelector("#account-profile:not([hidden])");
assert.match(norm(await page.locator("#profile-name").innerText()), /Penguji Baru/);
assert.match(norm(await page.locator("#profile-email").innerText()), /penguji@example\.com/);
assert.match(norm(await page.locator("#profile-role").innerText()), /Pembeli/);
assert.equal(await page.locator("#account-empty").isVisible(), true);
await page.screenshot({ path: path.join(OUT, "account-empty.png"), fullPage: true, animations: "disabled" });

for (const size of [
  { width: 768, height: 1024 },
  { width: 390, height: 844 },
  { width: 320, height: 740 },
]) {
  await page.setViewportSize(size);
  await page.reload({ waitUntil: "networkidle" });
  await page.waitForSelector("#account-profile:not([hidden])");
  await checkBasics(page, `account ${size.width}`);
}
await page.setViewportSize({ width: 390, height: 844 });
await page.screenshot({ path: path.join(OUT, "account-mobile.png"), fullPage: true, animations: "disabled" });
await page.setViewportSize({ width: 1440, height: 1100 });

await page.locator("#logout-account").click();
await page.waitForURL(/index\.html/);
await page.waitForSelector('#main[aria-busy="false"]');
assert.equal(await page.locator("#identity-guest").isVisible(), true);

// Duplicate email is rejected.
await page.goto(`${BASE_URL}register.html`, { waitUntil: "networkidle" });
await page.locator("#name").fill("Duplikat");
await page.locator("#email").fill("raihan@example.com");
await page.locator("#password").fill("rahasia123");
await page.locator("#confirm-password").fill("rahasia123");
await page.locator("#register-submit").click();
await page.waitForSelector("#register-error:not([hidden])");
assert.match(norm(await page.locator("#register-error").innerText()), /Email sudah terdaftar/);

// A bidder who opens admin sees the access gate.
await page.goto(`${BASE_URL}login.html?next=index.html`, { waitUntil: "networkidle" });
await page.waitForSelector('[data-demo-email="raihan@example.com"]');
await page.locator('[data-demo-email="raihan@example.com"]').click();
await page.waitForURL(/index\.html/);
await page.goto(`${BASE_URL}admin.html`, { waitUntil: "networkidle" });
await page.waitForSelector("#admin-gate:not([hidden])");
assert.match(norm(await page.locator("#gate-message").innerText()), /akun pembeli/);
assert.equal(await page.locator("#admin-main").isVisible(), false);
await page.screenshot({ path: path.join(OUT, "admin-gate-bidder.png"), fullPage: true, animations: "disabled" });

await page.setViewportSize({ width: 320, height: 740 });
await page.reload({ waitUntil: "networkidle" });
await page.waitForSelector("#admin-gate:not([hidden])");
await checkBasics(page, "admin gate 320");
await page.screenshot({ path: path.join(OUT, "admin-gate-320.png"), fullPage: true, animations: "disabled" });
await page.setViewportSize({ width: 1440, height: 1100 });

// Results archive is empty before any auction ends.
await page.goto(`${BASE_URL}results.html`, { waitUntil: "networkidle" });
await page.waitForSelector('#results-main[aria-busy="false"]');
assert.equal(await page.locator("#results-empty").isVisible(), true);

// The login page shows the current session and can clear it.
await page.goto(`${BASE_URL}login.html`, { waitUntil: "networkidle" });
await page.waitForSelector("#session-panel:not([hidden])");
assert.match(norm(await page.locator("#session-name").innerText()), /Raihan/);
assert.equal(await page.locator("#login-form").isVisible(), false);
await page.locator("#logout-current").click();
await page.waitForSelector("#login-form:not([hidden])");

// Responsive checks for the remaining account pages.
for (const size of [
  { width: 768, height: 1024 },
  { width: 390, height: 844 },
  { width: 320, height: 740 },
]) {
  await page.setViewportSize(size);

  await page.goto(`${BASE_URL}login.html`, { waitUntil: "networkidle" });
  await page.waitForSelector("#login-form:not([hidden])");
  await checkBasics(page, `login ${size.width}`);

  await page.goto(`${BASE_URL}register.html`, { waitUntil: "networkidle" });
  await page.waitForSelector("#register-form:not([hidden])");
  await checkBasics(page, `register ${size.width}`);

  await page.goto(`${BASE_URL}results.html`, { waitUntil: "networkidle" });
  await page.waitForSelector('#results-main[aria-busy="false"]');
  await checkBasics(page, `results ${size.width}`);
}
await page.setViewportSize({ width: 390, height: 844 });
await page.goto(`${BASE_URL}login.html`, { waitUntil: "networkidle" });
await page.waitForSelector("#login-form:not([hidden])");
await page.screenshot({ path: path.join(OUT, "login-mobile.png"), fullPage: true, animations: "disabled" });
await page.setViewportSize({ width: 1440, height: 1100 });

assert.deepEqual(errors, [], `console errors: ${errors.join(" | ")}`);
assert.deepEqual(pageErrors, [], `page errors: ${pageErrors.join(" | ")}`);

await browser.close();
console.log(
  JSON.stringify(
    {
      status: "PASS",
      checks: [
        "guest can watch but not bid",
        "login page validates empty and wrong password",
        "login form redirects with Enter",
        "header logout returns to guest",
        "register validates name and password confirmation",
        "register redirects to account page",
        "account page shows profile and empty bids",
        "duplicate email rejected",
        "bidder sees admin access gate",
        "results archive empty state",
        "login page detects existing session and logs out",
        "auth, account, and results pages have no overflow at 768/390/320",
        "44px targets and labels on auth pages",
        "zero console and page errors",
      ],
      screenshots: OUT,
    },
    null,
    2,
  ),
);
