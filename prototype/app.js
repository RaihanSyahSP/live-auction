const STORAGE_KEY = "lelang-demo-state-v1";
const NAME_KEY = "lelang-demo-name";
const CHANNEL_NAME = "lelang-demo-channel";
const DEMO_DURATION_MS = 10 * 60 * 1000;
const ANTI_SNIPE_MS = 60 * 1000;
const MAX_EXTENSIONS = 5;
const MAX_AMOUNT = 1_000_000_000_000;
const LOCK_NAME = "lelang-demo-lock";

const TAB_ID =
  typeof crypto !== "undefined" && crypto.randomUUID
    ? crypto.randomUUID()
    : `tab-${Math.random().toString(36).slice(2)}`;

const idr = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});
const numberFormat = new Intl.NumberFormat("id-ID");
const timeFormat = new Intl.DateTimeFormat("id-ID", {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

const refs = {
  main: document.getElementById("main"),
  banner: document.getElementById("app-banner"),
  liveState: document.getElementById("live-state"),
  liveLabel: document.getElementById("live-label"),
  priceLabel: document.getElementById("price-label"),
  price: document.getElementById("price"),
  priceMeta: document.getElementById("price-meta"),
  countdown: document.getElementById("countdown"),
  extensionNote: document.getElementById("extension-note"),
  timeFill: document.getElementById("time-fill"),
  identitySummary: document.getElementById("identity-summary"),
  identityName: document.getElementById("identity-name"),
  changeName: document.getElementById("change-name"),
  nameForm: document.getElementById("name-form"),
  nameInput: document.getElementById("name-input"),
  nameError: document.getElementById("name-error"),
  bidForm: document.getElementById("bid-form"),
  amountInput: document.getElementById("amount-input"),
  minimumBid: document.getElementById("minimum-bid"),
  increment: document.getElementById("increment"),
  quickBids: document.getElementById("quick-bids"),
  bidSubmit: document.getElementById("bid-submit"),
  bidError: document.getElementById("bid-error"),
  bidStatus: document.getElementById("bid-status"),
  result: document.getElementById("result"),
  resultTitle: document.getElementById("result-title"),
  resultPrice: document.getElementById("result-price"),
  resetResult: document.getElementById("reset-result"),
  bidCount: document.getElementById("bid-count"),
  ledgerList: document.getElementById("ledger-list"),
  ledgerEmpty: document.getElementById("ledger-empty"),
  ledgerLoading: document.getElementById("ledger-loading"),
  syncNote: document.getElementById("sync-note"),
  simulateRival: document.getElementById("simulate-rival"),
  resetDemo: document.getElementById("reset-demo"),
  toast: document.getElementById("toast"),
  toastMessage: document.getElementById("toast-message"),
  toastClose: document.getElementById("toast-close"),
  priceAnnouncer: document.getElementById("price-announcer"),
};

let currentState = null;
let currentName = "";
let editingName = false;
let pendingBid = false;
let amountValue = 0;
let amountTouched = false;
let channel = null;
let memoryState = null;
let storageAvailable = true;
let toastTimer = null;
let priceFlashTimer = null;
let ticking = false;
const peers = new Map();
const bannerMessages = new Set();

function seedState(withBids = true) {
  const now = Date.now();
  const bids = withBids
    ? [
        {
          id: 3,
          name: "Peserta 03",
          amount: 7_200_000,
          at: new Date(now - 60_000).toISOString(),
        },
        {
          id: 2,
          name: "Peserta 02",
          amount: 6_900_000,
          at: new Date(now - 120_000).toISOString(),
        },
        {
          id: 1,
          name: "Peserta 01",
          amount: 6_700_000,
          at: new Date(now - 180_000).toISOString(),
        },
      ]
    : [];

  return {
    version: 1,
    auction: {
      lot: "001",
      title: "Sony Alpha A7 II",
      subtitle: "Body only",
      startingPrice: 6_500_000,
      minIncrement: 100_000,
      currentPrice: withBids ? 7_200_000 : 6_500_000,
      status: "live",
      startsAt: new Date(now - (withBids ? 180_000 : 0)).toISOString(),
      endsAt: new Date(now + DEMO_DURATION_MS).toISOString(),
      extensionCount: 0,
      highestBidder: withBids ? "Peserta 03" : "",
      winner: "",
      finalPrice: null,
    },
    bids,
  };
}

function isValidState(value) {
  if (!value || value.version !== 1) return false;
  const auction = value.auction;
  if (!auction || !Array.isArray(value.bids)) return false;
  if (typeof auction.currentPrice !== "number") return false;
  if (typeof auction.startingPrice !== "number") return false;
  if (typeof auction.minIncrement !== "number") return false;
  if (!["live", "ended"].includes(auction.status)) return false;
  if (Number.isNaN(Date.parse(auction.startsAt))) return false;
  if (Number.isNaN(Date.parse(auction.endsAt))) return false;
  return value.bids.every(
    (bid) =>
      bid &&
      typeof bid.id === "number" &&
      typeof bid.name === "string" &&
      typeof bid.amount === "number" &&
      !Number.isNaN(Date.parse(bid.at)),
  );
}

function readState() {
  let raw = null;
  try {
    raw = localStorage.getItem(STORAGE_KEY);
  } catch {
    storageAvailable = false;
    return {
      state: memoryState || seedState(),
      fresh: !memoryState,
      recovered: false,
    };
  }

  if (!raw) {
    if (memoryState) return { state: memoryState, fresh: false, recovered: false };
    return { state: seedState(), fresh: true, recovered: false };
  }

  try {
    const parsed = JSON.parse(raw);
    if (isValidState(parsed)) {
      memoryState = parsed;
      return { state: parsed, fresh: false, recovered: false };
    }
  } catch {
    // Invalid JSON is treated as a recoverable demo state.
  }

  return { state: seedState(), fresh: false, recovered: true };
}

function writeState(next) {
  memoryState = next;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    storageAvailable = false;
    showBanner("Penyimpanan browser tidak tersedia. Sesi hanya bertahan di tab ini.");
  }
  channel?.postMessage({ type: "state", state: next, sender: TAB_ID });
  return next;
}

async function withStateLock(fn) {
  if (navigator.locks && typeof navigator.locks.request === "function") {
    return navigator.locks.request(LOCK_NAME, fn);
  }
  return fn();
}

function getMinimum(state) {
  return state.bids.length > 0
    ? state.auction.currentPrice + state.auction.minIncrement
    : state.auction.startingPrice;
}

function nextBidId(state) {
  return state.bids.reduce((max, bid) => Math.max(max, bid.id), 0) + 1;
}

function endAuction(state) {
  if (state.auction.status !== "live") return;
  state.auction.status = "ended";
  if (state.auction.highestBidder) {
    state.auction.winner = state.auction.highestBidder;
    state.auction.finalPrice = state.auction.currentPrice;
  } else {
    state.auction.winner = "";
    state.auction.finalPrice = null;
  }
}

async function initState() {
  return withStateLock(async () => {
    const loaded = readState();
    if (loaded.fresh || loaded.recovered) {
      const next = seedState(true);
      writeState(next);
      return { state: next, recovered: loaded.recovered };
    }
    if (
      loaded.state.auction.status === "live" &&
      Date.now() >= Date.parse(loaded.state.auction.endsAt)
    ) {
      endAuction(loaded.state);
      writeState(loaded.state);
    }
    return { state: loaded.state, recovered: false };
  });
}

async function placeBid({ name, amount }) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;

    if (state.auction.status !== "live") {
      return { ok: false, error: "Lelang sudah selesai." };
    }

    const now = Date.now();
    if (now >= Date.parse(state.auction.endsAt)) {
      endAuction(state);
      writeState(state);
      return { ok: false, error: "Lelang sudah selesai.", state };
    }

    if (!name || name.length < 2) {
      return { ok: false, error: "Isi nama dulu sebelum bid." };
    }
    if (!Number.isSafeInteger(amount) || amount <= 0) {
      return { ok: false, error: "Nominal bid tidak valid." };
    }
    if (amount > MAX_AMOUNT) {
      return { ok: false, error: "Nominal bid terlalu besar." };
    }
    if (state.auction.highestBidder === name) {
      return { ok: false, error: "Kamu sudah memegang harga tertinggi." };
    }

    const minimum = getMinimum(state);
    if (amount < minimum) {
      return {
        ok: false,
        error: `Bid minimum ${idr.format(minimum)}.`,
        minimum,
      };
    }

    const bid = {
      id: nextBidId(state),
      name,
      amount,
      at: new Date(now).toISOString(),
    };
    state.bids.unshift(bid);
    state.auction.currentPrice = amount;
    state.auction.highestBidder = name;

    let extended = false;
    const remaining = Date.parse(state.auction.endsAt) - now;
    if (remaining <= ANTI_SNIPE_MS && state.auction.extensionCount < MAX_EXTENSIONS) {
      state.auction.endsAt = new Date(
        Date.parse(state.auction.endsAt) + 60_000,
      ).toISOString();
      state.auction.extensionCount += 1;
      extended = true;
    }

    writeState(state);
    return { ok: true, state, bid, extended };
  });
}

async function resetDemo() {
  if (!window.confirm("Reset lelang? Semua bid akan dihapus.")) return;
  const next = await withStateLock(async () => {
    const fresh = seedState(false);
    writeState(fresh);
    return fresh;
  });
  pendingBid = false;
  editingName = false;
  amountTouched = false;
  refs.bidError.hidden = true;
  refs.bidStatus.hidden = true;
  applyState(next);
  showToast("Lelang demo dimulai ulang.");
}

function applyState(next) {
  if (!isValidState(next)) return;
  const previous = currentState;
  const previousHighest = previous?.auction.highestBidder;
  const previousPrice = previous?.auction.currentPrice;
  const previousExtensions = previous?.auction.extensionCount ?? 0;
  const previousBidCount = previous?.bids.length ?? 0;

  currentState = next;

  if (
    previous &&
    next.auction.status === "live" &&
    previousHighest === currentName &&
    next.auction.highestBidder &&
    next.auction.highestBidder !== currentName &&
    next.bids.length > previousBidCount
  ) {
    refs.bidStatus.hidden = true;
    showToast(
      `Kamu di-outbid. Harga baru ${idr.format(next.auction.currentPrice)} oleh ${next.auction.highestBidder}.`,
    );
  } else if (previous && next.auction.extensionCount > previousExtensions) {
    showToast("Waktu diperpanjang 60 detik.");
  }

  if (previous && next.auction.currentPrice !== previousPrice) {
    flashPrice();
    refs.priceAnnouncer.textContent = `Harga baru ${idr.format(
      next.auction.currentPrice,
    )} oleh ${next.auction.highestBidder}.`;
  }

  render();
}

function render() {
  if (!currentState) return;
  const auction = currentState.auction;
  const isLive = auction.status === "live";
  const hasBids = currentState.bids.length > 0;

  refs.liveState.classList.toggle("is-ended", !isLive);
  refs.liveLabel.textContent = isLive ? "LIVE" : "SELESAI";
  refs.priceLabel.textContent = hasBids ? "Harga saat ini" : "Harga awal";
  refs.price.textContent = idr.format(auction.currentPrice);
  renderMeta();

  refs.extensionNote.hidden = auction.extensionCount === 0;
  refs.extensionNote.textContent =
    auction.extensionCount > 0
      ? `Diperpanjang ${auction.extensionCount} dari ${MAX_EXTENSIONS} kali`
      : "";

  const hasName = currentName.length > 0;
  refs.identitySummary.hidden = !hasName || editingName;
  refs.nameForm.hidden = hasName && !editingName;
  refs.identityName.textContent = currentName;
  if (!hasName || editingName) {
    if (document.activeElement !== refs.nameInput) {
      refs.nameInput.value = currentName;
    }
  }

  refs.bidForm.hidden = !isLive;
  refs.result.hidden = isLive;
  refs.simulateRival.disabled = !isLive;
  refs.resetDemo.disabled = pendingBid;
  refs.resetResult.disabled = pendingBid;

  if (isLive) {
    const minimum = getMinimum(currentState);
    refs.minimumBid.textContent = idr.format(minimum);
    refs.increment.textContent = idr.format(auction.minIncrement);

    if (!amountTouched || amountValue < minimum) {
      amountValue = minimum;
      amountTouched = false;
      refs.amountInput.value = numberFormat.format(minimum);
    }

    updateQuickBids(minimum, auction.minIncrement);
    updateBidButton(minimum);
  } else {
    renderResult();
  }

  refs.bidCount.textContent = `${currentState.bids.length} bid`;
  refs.ledgerLoading.hidden = true;
  refs.ledgerEmpty.textContent = isLive
    ? "Belum ada bid. Bid pertama akan muncul di sini."
    : "Tidak ada bid pada lelang ini.";
  refs.ledgerEmpty.hidden = hasBids;
  refs.ledgerList.replaceChildren(
    ...currentState.bids.map((bid) => {
      const item = document.createElement("li");
      item.className = "ledger-item";

      const time = document.createElement("span");
      time.className = "ledger-time";
      time.textContent = formatTime(bid.at);

      const name = document.createElement("span");
      name.className = "ledger-name";
      name.textContent = bid.name;

      const amount = document.createElement("span");
      amount.className = "ledger-amount";
      amount.textContent = idr.format(bid.amount);

      item.append(time, name, amount);
      return item;
    }),
  );

  renderCountdown();
}

function renderMeta() {
  if (!currentState) return;
  const auction = currentState.auction;
  refs.priceMeta.textContent = `${auction.highestBidder || "Belum ada bid"} · ${
    currentState.bids.length
  } bid · ${viewerCount()} penonton`;
}

function renderCountdown() {
  if (!currentState) return;
  const auction = currentState.auction;
  if (auction.status !== "live") {
    refs.countdown.textContent = "Selesai";
    refs.countdown.setAttribute("aria-label", "Lelang selesai");
    refs.timeFill.style.width = "0%";
    return;
  }

  const now = Date.now();
  const end = Date.parse(auction.endsAt);
  const start = Date.parse(auction.startsAt);
  const remaining = Math.max(0, end - now);
  const total = Math.max(1, end - start);
  const percent = Math.max(0, Math.min(100, (remaining / total) * 100));
  const totalSeconds = Math.ceil(remaining / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  refs.countdown.textContent = `${String(minutes).padStart(2, "0")}:${String(
    seconds,
  ).padStart(2, "0")}`;
  refs.countdown.setAttribute(
    "aria-label",
    `Sisa waktu ${minutes} menit ${seconds} detik`,
  );
  refs.timeFill.style.width = `${percent}%`;
}

function renderResult() {
  const auction = currentState.auction;
  const winner = auction.winner;
  refs.resultTitle.textContent = winner
    ? `${winner} memenangkan lot ini.`
    : "Tidak ada bid. Lot tidak terjual.";
  refs.resultPrice.textContent =
    winner && typeof auction.finalPrice === "number"
      ? `Harga akhir ${idr.format(auction.finalPrice)}`
      : "Tidak ada harga akhir.";
}

function updateQuickBids(minimum, increment) {
  const amounts = [minimum, minimum + increment, minimum + increment * 4];
  const buttons = refs.quickBids.querySelectorAll("button");
  buttons.forEach((button, index) => {
    const amount = amounts[index];
    button.dataset.amount = String(amount);
    button.textContent = numberFormat.format(amount);
    button.setAttribute("aria-label", `Isi bid ${idr.format(amount)}`);
    button.disabled = currentState.auction.status !== "live";
  });
}

function updateBidButton(minimum) {
  if (!currentState) return;
  const isLive = currentState.auction.status === "live";
  let label = `Pasang bid ${idr.format(amountValue || minimum)}`;
  if (pendingBid) label = "Mengirim bid...";
  else if (!currentName) label = "Isi nama untuk bid";
  else if (!isLive) label = "Lelang selesai";

  refs.bidSubmit.textContent = label;
  refs.bidSubmit.disabled = !isLive || pendingBid || !currentName;
}

function viewerCount() {
  return 1 + peers.size;
}

function flashPrice() {
  clearTimeout(priceFlashTimer);
  refs.price.classList.remove("is-changed");
  void refs.price.offsetWidth;
  refs.price.classList.add("is-changed");
  priceFlashTimer = setTimeout(() => {
    refs.price.classList.remove("is-changed");
  }, 750);
}

function formatTime(value) {
  return timeFormat.format(new Date(value)).replaceAll(".", ":");
}

function showBidError(message) {
  refs.bidError.textContent = message;
  refs.bidError.hidden = false;
  refs.bidStatus.hidden = true;
}

function showNameError(message) {
  refs.nameError.textContent = message;
  refs.nameError.hidden = false;
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

function postPing() {
  channel?.postMessage({ type: "ping", id: TAB_ID, at: Date.now() });
}

function postHello() {
  channel?.postMessage({ type: "hello", id: TAB_ID });
}

function prunePeers() {
  const cutoff = Date.now() - 6000;
  for (const [id, seenAt] of peers) {
    if (seenAt < cutoff) peers.delete(id);
  }
}

function sayGoodbye() {
  channel?.postMessage({ type: "bye", id: TAB_ID });
}

function handleChannelMessage(event) {
  const message = event.data;
  if (!message || message.sender === TAB_ID || message.id === TAB_ID) return;

  if (message.type === "state" && isValidState(message.state)) {
    applyState(message.state);
    return;
  }
  if (message.type === "hello") {
    postPing();
    return;
  }
  if (message.type === "ping" && message.id) {
    peers.set(message.id, Date.now());
    renderMeta();
    return;
  }
  if (message.type === "bye" && message.id) {
    peers.delete(message.id);
    renderMeta();
  }
}

function setupQuickBids() {
  for (const index of [0, 1, 2]) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "button button-secondary";
    button.addEventListener("click", () => {
      amountValue = Number(button.dataset.amount);
      amountTouched = true;
      refs.amountInput.value = numberFormat.format(amountValue);
      refs.bidError.hidden = true;
      updateBidButton(getMinimum(currentState));
      refs.amountInput.focus();
    });
    refs.quickBids.append(button);
  }
}

function setupEvents() {
  refs.nameForm.addEventListener("submit", (event) => {
    event.preventDefault();
    const value = refs.nameInput.value.trim();
    if (value.length < 2) {
      showNameError("Nama minimal 2 karakter.");
      refs.nameInput.focus();
      return;
    }
    if (value.length > 40) {
      showNameError("Nama maksimal 40 karakter.");
      refs.nameInput.focus();
      return;
    }
    currentName = value;
    editingName = false;
    refs.nameError.hidden = true;
    refs.bidError.hidden = true;
    try {
      sessionStorage.setItem(NAME_KEY, value);
    } catch {
      showBanner("Nama tidak bisa disimpan di tab ini, tetapi tetap dipakai selama halaman terbuka.");
    }
    render();
    refs.amountInput.focus();
  });

  refs.nameInput.addEventListener("input", () => {
    refs.nameError.hidden = true;
  });

  refs.changeName.addEventListener("click", () => {
    editingName = true;
    render();
    refs.nameInput.focus();
    refs.nameInput.select();
  });

  refs.amountInput.addEventListener("focus", () => {
    refs.amountInput.select();
  });

  refs.amountInput.addEventListener("input", () => {
    const digits = refs.amountInput.value.replace(/\D/g, "").slice(0, 13);
    amountValue = digits ? Number(digits) : 0;
    amountTouched = true;
    refs.amountInput.value = digits ? numberFormat.format(amountValue) : "";
    refs.bidError.hidden = true;
    updateBidButton(getMinimum(currentState));
  });

  refs.bidForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (pendingBid) return;

    refs.bidError.hidden = true;
    refs.bidStatus.hidden = true;

    if (!currentName) {
      editingName = true;
      render();
      showNameError("Isi nama dulu sebelum bid.");
      refs.nameInput.focus();
      return;
    }
    if (currentState.auction.status !== "live") {
      showBidError("Lelang sudah selesai.");
      return;
    }

    const minimum = getMinimum(currentState);
    if (!Number.isSafeInteger(amountValue) || amountValue <= 0) {
      showBidError("Masukkan nominal bid yang valid.");
      return;
    }
    if (amountValue > MAX_AMOUNT) {
      showBidError("Nominal bid terlalu besar.");
      return;
    }
    if (amountValue < minimum) {
      amountValue = minimum;
      amountTouched = false;
      refs.amountInput.value = numberFormat.format(minimum);
      showBidError(`Bid minimum ${idr.format(minimum)}.`);
      updateBidButton(minimum);
      return;
    }

    pendingBid = true;
    render();

    const result = await placeBid({ name: currentName, amount: amountValue });
    pendingBid = false;

    if (result.ok) {
      applyState(result.state);
      refs.bidStatus.textContent = result.extended
        ? "Bid tercatat. Waktu diperpanjang 60 detik. Kamu memegang harga tertinggi."
        : "Bid tercatat. Kamu memegang harga tertinggi.";
      refs.bidStatus.hidden = false;
      amountTouched = false;
      amountValue = getMinimum(result.state);
      refs.amountInput.value = numberFormat.format(amountValue);
      render();
      return;
    }

    if (result.minimum) {
      amountValue = result.minimum;
      amountTouched = false;
      refs.amountInput.value = numberFormat.format(result.minimum);
    }
    if (result.state) applyState(result.state);
    showBidError(result.error || "Bid gagal. Coba lagi.");
    render();
  });

  refs.resetDemo.addEventListener("click", resetDemo);
  refs.resetResult.addEventListener("click", resetDemo);

  refs.simulateRival.addEventListener("click", async () => {
    if (!currentState || currentState.auction.status !== "live") return;
    const wasHighest = currentState.auction.highestBidder === currentName;
    const minimum = getMinimum(currentState);
    const rivalName =
      currentState.auction.highestBidder === "Penantang demo"
        ? "Penantang demo 2"
        : "Penantang demo";
    const result = await placeBid({ name: rivalName, amount: minimum });
    if (!result.ok) {
      showToast(result.error || "Simulasi penantang gagal.");
      return;
    }
    applyState(result.state);
    if (!wasHighest) {
      showToast(`${rivalName} memasang bid ${idr.format(minimum)}.`);
    }
  });

  refs.toastClose.addEventListener("click", hideToast);

  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) {
      postPing();
      tick();
    }
  });

  window.addEventListener("pagehide", sayGoodbye);
  window.addEventListener("beforeunload", () => {
    sayGoodbye();
    channel?.close();
  });
}

async function tick() {
  if (!currentState || ticking) return;
  renderCountdown();
  if (currentState.auction.status !== "live") return;
  if (Date.now() < Date.parse(currentState.auction.endsAt)) return;

  ticking = true;
  try {
    const ended = await withStateLock(async () => {
      const loaded = readState();
      if (
        loaded.state.auction.status === "live" &&
        Date.now() >= Date.parse(loaded.state.auction.endsAt)
      ) {
        endAuction(loaded.state);
        writeState(loaded.state);
        return loaded.state;
      }
      return null;
    });
    if (ended) applyState(ended);
  } finally {
    ticking = false;
  }
}

async function init() {
  try {
    const storedName = sessionStorage.getItem(NAME_KEY)?.trim() || "";
    currentName = storedName.length >= 2 ? storedName : "";
  } catch {
    currentName = "";
  }

  setupQuickBids();
  setupEvents();

  try {
    channel = new BroadcastChannel(CHANNEL_NAME);
    channel.addEventListener("message", handleChannelMessage);
  } catch {
    channel = null;
  }

  window.addEventListener("storage", (event) => {
    if (event.key !== STORAGE_KEY || !event.newValue) return;
    try {
      const parsed = JSON.parse(event.newValue);
      if (isValidState(parsed)) applyState(parsed);
    } catch {
      showBanner("State dari tab lain tidak bisa dibaca.");
    }
  });

  let loaded;
  try {
    loaded = await initState();
  } catch {
    currentState = seedState(true);
    try {
      writeState(currentState);
    } catch {
      // State stays in memory when storage is blocked.
    }
    showBanner("Sesi demo tidak bisa dimuat. Sesi baru dibuat.");
    loaded = { state: currentState, recovered: false };
  }
  currentState = loaded.state;

  if (loaded.recovered) {
    showBanner("State demo tidak terbaca. Sesi dimulai ulang.");
  }
  if (!storageAvailable) {
    showBanner("Penyimpanan browser tidak tersedia. Sesi hanya bertahan di tab ini.");
  }
  if (!channel) {
    showBanner(
      "Sinkronisasi antar tab tidak tersedia di browser ini. Sesi berjalan di tab ini saja.",
    );
  }

  render();
  refs.main.setAttribute("aria-busy", "false");
  refs.syncNote.textContent = channel
    ? "Sinkron antar tab aktif di browser ini."
    : "Sesi berjalan di tab ini saja.";
  refs.syncNote.classList.toggle("is-warning", !channel);

  postHello();
  postPing();
  tick();

  setInterval(tick, 250);
  setInterval(() => {
    postPing();
    prunePeers();
    renderMeta();
  }, 3000);
}

init();
