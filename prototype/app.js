import {
  MAX_AMOUNT,
  MAX_EXTENSIONS,
  NAME_KEY,
  TAB_ID,
  broadcast,
  connectStateChannel,
  endAuctionById,
  findAuction,
  getLiveAuction,
  getMinimum,
  idr,
  initState,
  isChannelAvailable,
  isStorageAvailable,
  numberFormat,
  placeBid,
  resetDemo,
} from "./store.js";

const ledgerTimeFormat = new Intl.DateTimeFormat("id-ID", {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

const requestedAuctionId = (() => {
  const raw = new URLSearchParams(window.location.search).get("auction");
  const value = Number(raw);
  return Number.isFinite(value) && value > 0 ? value : null;
})();

const refs = {
  main: document.getElementById("main"),
  banner: document.getElementById("app-banner"),
  lotSheet: document.getElementById("lot-sheet"),
  auctionSection: document.getElementById("auction-section"),
  noAuction: document.getElementById("no-auction"),
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
  previewNote: document.getElementById("preview-note"),
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
let currentAuction = null;
let selectedAuctionId = null;
let currentName = "";
let editingName = false;
let pendingBid = false;
let amountValue = 0;
let amountTouched = false;
let toastTimer = null;
let priceFlashTimer = null;
let ticking = false;
const peers = new Map();
const bannerMessages = new Set();

function selectAuction(state) {
  if (requestedAuctionId !== null) {
    selectedAuctionId = findAuction(state, requestedAuctionId)?.id ?? null;
  } else {
    selectedAuctionId = getLiveAuction(state)?.id ?? selectedAuctionId;
  }

  if (selectedAuctionId === null) {
    selectedAuctionId = state.auctions[0]?.id ?? null;
  }

  currentAuction = findAuction(state, selectedAuctionId);
  if (!currentAuction) {
    currentAuction = getLiveAuction(state) || state.auctions[0] || null;
    selectedAuctionId = currentAuction?.id ?? null;
  }
}

function applyState(next) {
  if (!next || !Array.isArray(next.auctions)) return;

  const previousAuction = currentAuction;
  currentState = next;
  selectAuction(next);

  if (!currentAuction) {
    renderEmpty();
    return;
  }

  if (previousAuction && previousAuction.id === currentAuction.id) {
    const previousHighest = previousAuction.highestBidder;
    const previousPrice = previousAuction.currentPrice;
    const previousExtensions = previousAuction.extensionCount;
    const previousBidCount = previousAuction.bids.length;

    if (
      currentAuction.status === "live" &&
      previousHighest === currentName &&
      currentAuction.highestBidder &&
      currentAuction.highestBidder !== currentName &&
      currentAuction.bids.length > previousBidCount
    ) {
      refs.bidStatus.hidden = true;
      showToast(
        `Kamu di-outbid. Harga baru ${idr.format(
          currentAuction.currentPrice,
        )} oleh ${currentAuction.highestBidder}.`,
      );
    } else if (currentAuction.extensionCount > previousExtensions) {
      showToast("Waktu diperpanjang 60 detik.");
    }

    if (currentAuction.currentPrice !== previousPrice) {
      flashPrice();
      refs.priceAnnouncer.textContent = `Harga baru ${idr.format(
        currentAuction.currentPrice,
      )} oleh ${currentAuction.highestBidder}.`;
    }
  }

  render();
}

function renderEmpty() {
  refs.noAuction.hidden = false;
  refs.lotSheet.hidden = true;
  refs.auctionSection.hidden = true;
  refs.liveState.classList.add("is-ended");
  refs.liveState.classList.remove("is-draft");
  refs.liveLabel.textContent = "KOSONG";
  refs.main.setAttribute("aria-busy", "false");
}

function render() {
  if (!currentAuction) {
    renderEmpty();
    return;
  }

  refs.noAuction.hidden = true;
  refs.lotSheet.hidden = false;
  refs.auctionSection.hidden = false;

  const auction = currentAuction;
  const isLive = auction.status === "live";
  const isDraft = auction.status === "draft";
  const isEnded = auction.status === "ended";
  const hasBids = auction.bids.length > 0;

  refs.liveState.classList.toggle("is-ended", isEnded);
  refs.liveState.classList.toggle("is-draft", isDraft);
  refs.liveLabel.textContent = isLive ? "LIVE" : isEnded ? "SELESAI" : "DRAFT";
  refs.priceLabel.textContent = hasBids ? "Harga saat ini" : "Harga awal";
  refs.price.textContent = idr.format(auction.currentPrice);
  renderMeta();

  refs.extensionNote.hidden = !isLive || auction.extensionCount === 0;
  refs.extensionNote.textContent =
    isLive && auction.extensionCount > 0
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
  refs.previewNote.hidden = !isDraft;
  refs.result.hidden = !isEnded;
  refs.simulateRival.disabled = !isLive;
  refs.resetDemo.disabled = pendingBid;
  refs.resetResult.disabled = pendingBid;

  if (isLive) {
    const minimum = getMinimum(auction);
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
    refs.bidError.hidden = true;
    refs.bidStatus.hidden = true;
  }

  if (isEnded) {
    renderResult();
  }

  refs.bidCount.textContent = `${auction.bids.length} bid`;
  refs.ledgerLoading.hidden = true;
  refs.ledgerEmpty.textContent = isEnded
    ? "Tidak ada bid pada lelang ini."
    : "Belum ada bid. Bid pertama akan muncul di sini.";
  refs.ledgerEmpty.hidden = hasBids;
  refs.ledgerList.replaceChildren(
    ...auction.bids.map((bid) => {
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
  if (!currentAuction) return;
  refs.priceMeta.textContent = `${currentAuction.highestBidder || "Belum ada bid"} · ${
    currentAuction.bids.length
  } bid · ${viewerCount()} penonton`;
}

function renderCountdown() {
  if (!currentAuction) return;
  const auction = currentAuction;

  if (auction.status === "draft") {
    refs.countdown.textContent = "Belum dimulai";
    refs.countdown.setAttribute("aria-label", "Lelang belum dimulai");
    refs.timeFill.style.width = "0%";
    return;
  }

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
  const winner = currentAuction.winner;
  refs.resultTitle.textContent = winner
    ? `${winner} memenangkan lot ini.`
    : "Tidak ada bid. Lot tidak terjual.";
  refs.resultPrice.textContent =
    winner && typeof currentAuction.finalPrice === "number"
      ? `Harga akhir ${idr.format(currentAuction.finalPrice)}`
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
    button.disabled = currentAuction?.status !== "live";
  });
}

function updateBidButton(minimum) {
  if (!currentAuction) return;
  const isLive = currentAuction.status === "live";
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
  return ledgerTimeFormat.format(new Date(value)).replaceAll(".", ":");
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
  broadcast({ type: "ping", id: TAB_ID });
}

function postHello() {
  broadcast({ type: "hello", id: TAB_ID });
}

function prunePeers() {
  const cutoff = Date.now() - 6000;
  for (const [id, seenAt] of peers) {
    if (seenAt < cutoff) peers.delete(id);
  }
}

function sayGoodbye() {
  broadcast({ type: "bye", id: TAB_ID });
}

function handlePeerMessage(message) {
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
  for (let index = 0; index < 3; index += 1) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "button button-secondary";
    button.addEventListener("click", () => {
      amountValue = Number(button.dataset.amount);
      amountTouched = true;
      refs.amountInput.value = numberFormat.format(amountValue);
      refs.bidError.hidden = true;
      updateBidButton(getMinimum(currentAuction));
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
      showBanner(
        "Nama tidak bisa disimpan di tab ini, tetapi tetap dipakai selama halaman terbuka.",
      );
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
    updateBidButton(getMinimum(currentAuction));
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
    if (!currentAuction || currentAuction.status !== "live") {
      showBidError("Lelang sudah selesai.");
      return;
    }

    const minimum = getMinimum(currentAuction);
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

    const result = await placeBid({
      auctionId: currentAuction.id,
      name: currentName,
      amount: amountValue,
    });
    pendingBid = false;

    if (result.ok) {
      applyState(result.state);
      refs.bidStatus.textContent = result.extended
        ? "Bid tercatat. Waktu diperpanjang 60 detik. Kamu memegang harga tertinggi."
        : "Bid tercatat. Kamu memegang harga tertinggi.";
      refs.bidStatus.hidden = false;
      amountTouched = false;
      amountValue = getMinimum(result.auction);
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

  refs.resetDemo.addEventListener("click", async () => {
    if (!window.confirm("Reset seluruh demo? Semua lelang dan bid akan dihapus.")) {
      return;
    }
    const next = await resetDemo();
    pendingBid = false;
    editingName = false;
    amountTouched = false;
    refs.bidError.hidden = true;
    refs.bidStatus.hidden = true;
    applyState(next);
    showToast("Demo dimulai ulang.");
  });

  refs.resetResult.addEventListener("click", async () => {
    if (!window.confirm("Reset seluruh demo? Semua lelang dan bid akan dihapus.")) {
      return;
    }
    const next = await resetDemo();
    pendingBid = false;
    editingName = false;
    amountTouched = false;
    refs.bidError.hidden = true;
    refs.bidStatus.hidden = true;
    applyState(next);
    showToast("Demo dimulai ulang.");
  });

  refs.simulateRival.addEventListener("click", async () => {
    if (!currentAuction || currentAuction.status !== "live") return;
    const wasHighest = currentAuction.highestBidder === currentName;
    const minimum = getMinimum(currentAuction);
    const rivalName =
      currentAuction.highestBidder === "Penantang demo"
        ? "Penantang demo 2"
        : "Penantang demo";
    const result = await placeBid({
      auctionId: currentAuction.id,
      name: rivalName,
      amount: minimum,
    });
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
  });
}

async function tick() {
  if (!currentAuction || ticking) return;
  renderCountdown();
  if (currentAuction.status !== "live") return;
  if (Date.now() < Date.parse(currentAuction.endsAt)) return;

  ticking = true;
  try {
    const result = await endAuctionById(currentAuction.id);
    if (result.ok) applyState(result.state);
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
  connectStateChannel({ onState: applyState, onMessage: handlePeerMessage });

  let loaded;
  try {
    loaded = await initState();
  } catch {
    currentState = { version: 2, auctions: [] };
    showBanner("Sesi demo tidak bisa dimuat. Coba muat ulang halaman.");
    loaded = { state: currentState, recovered: false };
  }

  currentState = loaded.state;
  selectAuction(currentState);

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

  render();
  refs.main.setAttribute("aria-busy", "false");
  refs.syncNote.textContent = isChannelAvailable()
    ? "Sinkron antar tab aktif di browser ini."
    : "Sesi berjalan di tab ini saja.";
  refs.syncNote.classList.toggle("is-warning", !isChannelAvailable());

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
