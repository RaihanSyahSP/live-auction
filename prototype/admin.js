import { getSession, loginDemo } from "./auth.js";
import { mountAuthLinks } from "./auth-ui.js";
import {
  MAX_AMOUNT,
  connectStateChannel,
  createDraft,
  deleteDraft,
  endAuctionById,
  getLiveAuction,
  idr,
  initState,
  isChannelAvailable,
  isStorageAvailable,
  numberFormat,
  resetDemo,
  startAuction,
  timeFormat,
} from "./store.js";

const refs = {
  main: document.getElementById("admin-main"),
  banner: document.getElementById("admin-banner"),
  gate: document.getElementById("admin-gate"),
  gateMessage: document.getElementById("gate-message"),
  adminDemoLogin: document.getElementById("admin-demo-login"),
  boot: document.getElementById("admin-boot"),
  footer: document.getElementById("admin-footer"),
  summary: document.getElementById("admin-summary"),
  loading: document.getElementById("admin-loading"),
  list: document.getElementById("admin-list"),
  empty: document.getElementById("admin-empty"),
  form: document.getElementById("create-form"),
  title: document.getElementById("auction-title"),
  subtitle: document.getElementById("auction-subtitle"),
  description: document.getElementById("auction-description"),
  condition: document.getElementById("auction-condition"),
  includes: document.getElementById("auction-includes"),
  specs: document.getElementById("auction-specs"),
  startingPrice: document.getElementById("starting-price"),
  minIncrement: document.getElementById("min-increment"),
  duration: document.getElementById("duration"),
  formError: document.getElementById("form-error"),
  formStatus: document.getElementById("form-status"),
  createSubmit: document.getElementById("create-submit"),
  resetAll: document.getElementById("reset-all"),
  toast: document.getElementById("toast"),
  toastMessage: document.getElementById("toast-message"),
  toastClose: document.getElementById("toast-close"),
};

let currentState = null;
let pendingAction = false;
let pendingCreate = false;
let toastTimer = null;
const bannerMessages = new Set();

function applyState(next) {
  if (!next || !Array.isArray(next.auctions)) return;
  currentState = next;
  render();
}

function render() {
  if (!currentState) return;

  const auctions = [...currentState.auctions].sort((a, b) => {
    const rank = { live: 0, draft: 1, ended: 2 };
    return rank[a.status] - rank[b.status] || b.id - a.id;
  });
  const live = auctions.filter((auction) => auction.status === "live");
  const drafts = auctions.filter((auction) => auction.status === "draft");
  const ended = auctions.filter((auction) => auction.status === "ended");
  const liveAuction = live[0] || null;

  refs.summary.textContent = `${live.length} live · ${drafts.length} draft · ${ended.length} selesai`;
  refs.loading.hidden = true;
  refs.empty.hidden = auctions.length > 0;
  refs.list.replaceChildren(
    ...auctions.map((auction) => renderCard(auction, liveAuction)),
  );
}

function renderCard(auction, liveAuction) {
  const article = document.createElement("article");
  article.className = `admin-card is-${auction.status}`;
  article.dataset.auctionId = String(auction.id);

  const top = document.createElement("div");
  top.className = "admin-card-top";

  const status = document.createElement("p");
  status.className = `admin-status is-${auction.status}`;
  status.textContent =
    auction.status === "live"
      ? "LIVE"
      : auction.status === "draft"
        ? "DRAFT"
        : "SELESAI";

  const lot = document.createElement("p");
  lot.className = "admin-lot";
  lot.textContent = `LOT ${auction.lot}`;

  top.append(status, lot);

  const title = document.createElement("h2");
  title.className = "admin-card-title";
  title.textContent = auction.title;

  article.append(top, title);

  if (auction.subtitle) {
    const subtitle = document.createElement("p");
    subtitle.className = "admin-card-subtitle";
    subtitle.textContent = auction.subtitle;
    article.append(subtitle);
  }

  const meta = document.createElement("dl");
  meta.className = "admin-meta";
  addMeta(meta, "Harga", idr.format(auction.currentPrice));
  addMeta(meta, "Bid", `${auction.bids.length} bid`);

  if (auction.status === "draft") {
    addMeta(meta, "Kelipatan", idr.format(auction.minIncrement));
    addMeta(meta, "Durasi", `${Math.round(auction.durationSeconds / 60)} menit`);
  } else if (auction.status === "live") {
    addMeta(meta, "Berakhir", timeFormat.format(new Date(auction.endsAt)));
  } else {
    addMeta(meta, "Pemenang", auction.winner || "Tidak terjual");
    addMeta(
      meta,
      "Harga akhir",
      typeof auction.finalPrice === "number"
        ? idr.format(auction.finalPrice)
        : "Tidak ada",
    );
  }

  article.append(meta);

  const actions = document.createElement("div");
  actions.className = "admin-actions";

  if (auction.status === "draft") {
    const preview = document.createElement("a");
    preview.className = "button button-secondary";
    preview.href = `index.html?auction=${auction.id}`;
    preview.textContent = "Preview";

    const start = document.createElement("button");
    start.type = "button";
    start.className = "button button-secondary";
    start.dataset.action = "start";
    start.dataset.id = String(auction.id);
    start.textContent = "Start lelang";
    start.disabled = Boolean(liveAuction);
    if (liveAuction) {
      start.title = "Akhiri lelang live dulu.";
    }

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "button button-danger";
    remove.dataset.action = "delete";
    remove.dataset.id = String(auction.id);
    remove.textContent = "Hapus";

    actions.append(preview, start, remove);
  } else if (auction.status === "live") {
    const open = document.createElement("a");
    open.className = "button button-secondary";
    open.href = `index.html?auction=${auction.id}`;
    open.textContent = "Buka ruang";

    const end = document.createElement("button");
    end.type = "button";
    end.className = "button button-danger";
    end.dataset.action = "end";
    end.dataset.id = String(auction.id);
    end.textContent = "End lelang";

    actions.append(open, end);
  } else {
    const result = document.createElement("a");
    result.className = "button button-secondary";
    result.href = `index.html?auction=${auction.id}`;
    result.textContent = "Lihat hasil";
    actions.append(result);
  }

  article.append(actions);
  return article;
}

function addMeta(list, label, value) {
  const wrapper = document.createElement("div");
  const term = document.createElement("dt");
  term.textContent = label;
  const description = document.createElement("dd");
  description.textContent = value;
  wrapper.append(term, description);
  list.append(wrapper);
}

function showBanner(message) {
  bannerMessages.add(message);
  refs.banner.textContent = [...bannerMessages].join(" ");
  refs.banner.hidden = false;
}

function showToast(message) {
  clearTimeout(toastTimer);
  refs.toastMessage.textContent = message;
  refs.toast.hidden = false;
  toastTimer = setTimeout(hideToast, 8000);
}

function hideToast() {
  clearTimeout(toastTimer);
  refs.toast.hidden = true;
}

function showFormError(message) {
  refs.formError.textContent = message;
  refs.formError.hidden = false;
  refs.formStatus.hidden = true;
}

function parsePrice(value) {
  const digits = String(value).replace(/\D/g, "");
  return digits ? Number(digits) : 0;
}

function attachPriceFormatting(input) {
  input.addEventListener("input", () => {
    const digits = input.value.replace(/\D/g, "").slice(0, 13);
    input.value = digits ? numberFormat.format(Number(digits)) : "";
  });
}

async function handleStart(id) {
  if (pendingAction) return;
  if (!window.confirm("Mulai lelang sekarang? Timer akan berjalan.")) return;
  pendingAction = true;
  try {
    const result = await startAuction(id);
    if (!result.ok) {
      showBanner(result.error);
      return;
    }
    applyState(result.state);
    showToast("Lelang dimulai. Buka ruang untuk melihatnya.");
  } finally {
    pendingAction = false;
  }
}

async function handleEnd(id) {
  if (pendingAction) return;
  if (!window.confirm("Akhiri lelang sekarang? Pemenang ditentukan dari bid tertinggi.")) {
    return;
  }
  pendingAction = true;
  try {
    const result = await endAuctionById(id);
    if (!result.ok) {
      showBanner(result.error);
      return;
    }
    applyState(result.state);
    showToast("Lelang diakhiri.");
  } finally {
    pendingAction = false;
  }
}

async function handleDelete(id) {
  if (pendingAction) return;
  if (!window.confirm("Hapus draft ini?")) return;
  pendingAction = true;
  try {
    const result = await deleteDraft(id);
    if (!result.ok) {
      showBanner(result.error);
      return;
    }
    applyState(result.state);
    showToast("Draft dihapus.");
  } finally {
    pendingAction = false;
  }
}

async function handleResetAll() {
  if (pendingAction) return;
  if (!window.confirm("Reset semua demo? Semua lelang dan bid akan dihapus.")) return;
  pendingAction = true;
  try {
    const state = await resetDemo({ withBids: true, withDraft: true });
    applyState(state);
    showToast("Demo dikembalikan ke data awal.");
  } finally {
    pendingAction = false;
  }
}

function setupEvents() {
  refs.list.addEventListener("click", (event) => {
    const button = event.target.closest("button[data-action]");
    if (!button) return;
    const id = Number(button.dataset.id);
    if (button.dataset.action === "start") handleStart(id);
    if (button.dataset.action === "end") handleEnd(id);
    if (button.dataset.action === "delete") handleDelete(id);
  });

  refs.form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (pendingCreate) return;

    refs.formError.hidden = true;
    refs.formStatus.hidden = true;

    const startingPrice = parsePrice(refs.startingPrice.value);
    const minIncrement = parsePrice(refs.minIncrement.value);
    const durationMinutes = Number(refs.duration.value);

    if (startingPrice > MAX_AMOUNT || minIncrement > MAX_AMOUNT) {
      showFormError("Nominal terlalu besar.");
      return;
    }

    pendingCreate = true;
    refs.createSubmit.disabled = true;
    refs.createSubmit.textContent = "Menyimpan...";

    const result = await createDraft({
      title: refs.title.value,
      subtitle: refs.subtitle.value,
      description: refs.description.value,
      condition: refs.condition.value,
      includes: refs.includes.value,
      specs: refs.specs.value,
      startingPrice,
      minIncrement,
      durationMinutes,
    });

    pendingCreate = false;
    refs.createSubmit.disabled = false;
    refs.createSubmit.textContent = "Simpan sebagai draft";

    if (!result.ok) {
      showFormError(result.error || "Draft gagal dibuat.");
      return;
    }

    applyState(result.state);
    refs.form.reset();
    refs.duration.value = "10";
    refs.formStatus.textContent =
      "Draft dibuat. Cari di daftar lelang, lalu tekan Start.";
    refs.formStatus.hidden = false;
    showToast("Draft dibuat.");
  });

  refs.resetAll.addEventListener("click", handleResetAll);
  refs.toastClose.addEventListener("click", hideToast);

  attachPriceFormatting(refs.startingPrice);
  attachPriceFormatting(refs.minIncrement);

  for (const link of document.querySelectorAll('a[href^="#"]')) {
    link.addEventListener("click", () => {
      const target = document.querySelector(link.getAttribute("href"));
      if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
        target.focus();
      }
    });
  }
}

async function tick() {
  if (!currentState) return;
  const live = getLiveAuction(currentState);
  if (!live || Date.now() < Date.parse(live.endsAt)) return;

  const result = await endAuctionById(live.id);
  if (result.ok) applyState(result.state);
}

async function init() {
  mountAuthLinks(document.getElementById("auth-links"), {
    logoutTarget: "index.html",
  });

  refs.adminDemoLogin.addEventListener("click", () => {
    const result = loginDemo("admin@example.com");
    if (!result.ok) {
      refs.gateMessage.textContent = result.error;
      return;
    }
    window.location.reload();
  });

  const session = getSession();
  if (!session || session.role !== "admin") {
    refs.boot.hidden = true;
    refs.gate.hidden = false;
    refs.main.hidden = true;
    refs.footer.hidden = true;
    refs.gateMessage.textContent = session
      ? `Kamu masuk sebagai ${session.name} dengan akun pembeli. Akun ini tidak punya akses admin.`
      : "Halaman ini memerlukan akun admin. Pakai akun demo di bawah untuk mencoba.";
    return;
  }

  refs.gate.hidden = true;
  refs.boot.hidden = true;
  refs.main.hidden = false;
  refs.footer.hidden = false;

  setupEvents();
  refs.duration.value = "10";
  connectStateChannel({ onState: applyState });

  let loaded;
  try {
    loaded = await initState();
  } catch {
    currentState = { version: 4, auctions: [] };
    showBanner("Sesi demo tidak bisa dimuat. Muat ulang halaman untuk mencoba lagi.");
    loaded = { state: currentState, recovered: false };
  }

  applyState(loaded.state);

  if (loaded.recovered) {
    showBanner("State demo tidak terbaca. Sesi dimulai ulang.");
  }
  if (!isStorageAvailable()) {
    showBanner(
      "Penyimpanan browser tidak tersedia. Sesi hanya bertahan di tab ini.",
    );
  }
  if (!isChannelAvailable()) {
    showBanner(
      "Sinkronisasi antar tab tidak tersedia di browser ini. Sesi berjalan di tab ini saja.",
    );
  }

  refs.main.setAttribute("aria-busy", "false");
  setInterval(tick, 1000);
}

init();
