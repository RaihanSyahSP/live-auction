export const STORAGE_KEY = "lelang-demo-state-v4";
export const CHANNEL_NAME = "lelang-demo-channel";
export const LOCK_NAME = "lelang-demo-lock";
export const MAX_EXTENSIONS = 5;
export const ANTI_SNIPE_MS = 60_000;
export const MAX_AMOUNT = 1_000_000_000_000;
export const DEMO_DURATION_MS = 10 * 60 * 1000;

export const TAB_ID =
  typeof crypto !== "undefined" && crypto.randomUUID
    ? crypto.randomUUID()
    : `tab-${Math.random().toString(36).slice(2)}`;

export const idr = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  maximumFractionDigits: 0,
});
export const numberFormat = new Intl.NumberFormat("id-ID");
export const timeFormat = new Intl.DateTimeFormat("id-ID", {
  hour: "2-digit",
  minute: "2-digit",
});

let channel = null;
let memoryState = null;
let storageAvailable = true;
let stateCallback = null;
let messageCallback = null;

export function isStorageAvailable() {
  return storageAvailable;
}

export function isChannelAvailable() {
  return Boolean(channel);
}

export function connectStateChannel({ onState, onMessage } = {}) {
  stateCallback = onState;
  messageCallback = onMessage;

  try {
    channel = new BroadcastChannel(CHANNEL_NAME);
    channel.addEventListener("message", (event) => {
      const message = event.data;
      if (!message || message.sender === TAB_ID) return;

      if (message.type === "state" && isValidState(message.state)) {
        memoryState = message.state;
        stateCallback?.(message.state);
        return;
      }

      messageCallback?.(message);
    });
  } catch {
    channel = null;
  }

  window.addEventListener("storage", (event) => {
    if (channel || event.key !== STORAGE_KEY || !event.newValue) return;
    try {
      const parsed = JSON.parse(event.newValue);
      if (isValidState(parsed)) {
        memoryState = parsed;
        stateCallback?.(parsed);
      }
    } catch {
      // Other tabs still get the state message through BroadcastChannel.
    }
  });
}

export function broadcast(message) {
  channel?.postMessage({ ...message, sender: TAB_ID });
}

function makeAuction({
  id,
  lot,
  title,
  subtitle,
  description,
  condition,
  includes,
  specs = [],
  startingPrice,
  minIncrement,
  durationSeconds,
  status,
  startsAt = null,
  endsAt = null,
}) {
  return {
    id,
    lot,
    title,
    subtitle,
    description,
    condition,
    includes,
    specs,
    startingPrice,
    minIncrement,
    durationSeconds,
    currentPrice: startingPrice,
    status,
    startsAt,
    endsAt,
    extensionCount: 0,
    highestBidder: "",
    highestBidderEmail: "",
    winner: "",
    winnerEmail: "",
    finalPrice: null,
    endedAt: null,
    bids: [],
  };
}

export function seedState({ withBids = true, withDraft = true } = {}) {
  const now = Date.now();
  const live = makeAuction({
    id: 1,
    lot: "001",
    title: "Sony Alpha A7 II",
    subtitle: "Body only",
    description:
      "Kamera full-frame 24 MP dengan stabilisasi 5-axis. Body saja, tanpa lensa. Dial dan tombol responsif, layar belakang punya baret halus. Baterai masih menyala normal.",
    condition: "Used, 8/10",
    includes: ["Body kamera", "Baterai NP-FW50", "Charger", "Tutup body"],
    specs: [
      { label: "Sensor", value: "Full-frame 24 MP" },
      { label: "Mount", value: "Sony E-mount" },
      { label: "Stabilisasi", value: "5-axis in-body" },
      { label: "Video", value: "1080p" },
    ],
    startingPrice: 6_500_000,
    minIncrement: 100_000,
    durationSeconds: 600,
    status: "live",
    startsAt: new Date(now - 180_000).toISOString(),
    endsAt: new Date(now + DEMO_DURATION_MS).toISOString(),
  });

  if (withBids) {
    live.bids = [
      {
        id: 3,
        name: "Peserta 03",
        email: "peserta03@demo.local",
        amount: 7_200_000,
        at: new Date(now - 60_000).toISOString(),
      },
      {
        id: 2,
        name: "Peserta 02",
        email: "peserta02@demo.local",
        amount: 6_900_000,
        at: new Date(now - 120_000).toISOString(),
      },
      {
        id: 1,
        name: "Peserta 01",
        email: "peserta01@demo.local",
        amount: 6_700_000,
        at: new Date(now - 180_000).toISOString(),
      },
    ];
    live.currentPrice = 7_200_000;
    live.highestBidder = "Peserta 03";
    live.highestBidderEmail = "peserta03@demo.local";
  }

  const auctions = [live];

  if (withDraft) {
    auctions.push(
      makeAuction({
        id: 2,
        lot: "002",
        title: "Fujifilm X-T2",
        subtitle: "Body only",
        description:
          "Mirrorless APS-C 24 MP dengan dial fisik dan layar lipat. Body saja, tanpa lensa. Kondisi mekanis normal, ada bekas pakai halus di bodi.",
        condition: "Used, 7/10",
        includes: ["Body kamera", "Baterai", "Charger"],
        specs: [
          { label: "Sensor", value: "APS-C 24 MP" },
          { label: "Mount", value: "Fujifilm X" },
          { label: "Stabilisasi", value: "Tidak ada" },
          { label: "Video", value: "4K" },
        ],
        startingPrice: 8_500_000,
        minIncrement: 100_000,
        durationSeconds: 600,
        status: "draft",
      }),
    );
  }

  return { version: 4, auctions };
}

function isValidBid(bid) {
  return (
    bid &&
    typeof bid.id === "number" &&
    typeof bid.name === "string" &&
    typeof bid.email === "string" &&
    typeof bid.amount === "number" &&
    !Number.isNaN(Date.parse(bid.at))
  );
}

function isValidAuction(auction) {
  if (!auction || typeof auction.id !== "number") return false;
  if (!["draft", "live", "ended"].includes(auction.status)) return false;
  if (typeof auction.title !== "string" || auction.title.trim().length === 0) return false;
  if (typeof auction.description !== "string") return false;
  if (typeof auction.condition !== "string") return false;
  if (!Array.isArray(auction.includes)) return false;
  if (
    !Array.isArray(auction.specs) ||
    !auction.specs.every(
      (spec) =>
        spec &&
        typeof spec.label === "string" &&
        typeof spec.value === "string",
    )
  ) {
    return false;
  }
  if (typeof auction.startingPrice !== "number") return false;
  if (typeof auction.minIncrement !== "number") return false;
  if (typeof auction.currentPrice !== "number") return false;
  if (typeof auction.durationSeconds !== "number") return false;
  if (typeof auction.extensionCount !== "number") return false;
  if (typeof auction.highestBidder !== "string") return false;
  if (typeof auction.highestBidderEmail !== "string") return false;
  if (typeof auction.winner !== "string") return false;
  if (typeof auction.winnerEmail !== "string") return false;
  if (auction.finalPrice !== null && typeof auction.finalPrice !== "number") return false;
  if (auction.endedAt && Number.isNaN(Date.parse(auction.endedAt))) return false;
  if (!Array.isArray(auction.bids) || !auction.bids.every(isValidBid)) return false;

  if (auction.status === "draft") return true;
  if (Number.isNaN(Date.parse(auction.startsAt))) return false;
  if (Number.isNaN(Date.parse(auction.endsAt))) return false;
  return true;
}

export function isValidState(value) {
  return (
    value &&
    value.version === 4 &&
    Array.isArray(value.auctions) &&
    value.auctions.every(isValidAuction)
  );
}

export function readState() {
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

export function writeState(state) {
  memoryState = state;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    storageAvailable = false;
  }
  broadcast({ type: "state", state });
  return state;
}

export async function withStateLock(fn) {
  if (navigator.locks && typeof navigator.locks.request === "function") {
    return navigator.locks.request(LOCK_NAME, fn);
  }
  return fn();
}

export function findAuction(state, id) {
  if (!state || id === null || id === undefined) return null;
  return state.auctions.find((auction) => auction.id === id) || null;
}

export function getLiveAuction(state) {
  return state?.auctions.find((auction) => auction.status === "live") || null;
}

export function getMinimum(auction) {
  return auction.bids.length > 0
    ? auction.currentPrice + auction.minIncrement
    : auction.startingPrice;
}

function nextBidId(auction) {
  return auction.bids.reduce((max, bid) => Math.max(max, bid.id), 0) + 1;
}

export function endAuction(auction) {
  if (auction.status !== "live") return;
  auction.status = "ended";
  auction.endedAt = new Date().toISOString();
  if (auction.highestBidder) {
    auction.winner = auction.highestBidder;
    auction.winnerEmail = auction.highestBidderEmail;
    auction.finalPrice = auction.currentPrice;
  } else {
    auction.winner = "";
    auction.winnerEmail = "";
    auction.finalPrice = null;
  }
}

export async function initState() {
  return withStateLock(async () => {
    const loaded = readState();
    let state = loaded.state;
    let changed = loaded.fresh || loaded.recovered;

    if (!state.auctions || state.auctions.length === 0) {
      state = seedState();
      changed = true;
    }

    const liveAuctions = state.auctions.filter((auction) => auction.status === "live");
    for (let index = 1; index < liveAuctions.length; index += 1) {
      endAuction(liveAuctions[index]);
      changed = true;
    }

    for (const auction of state.auctions) {
      if (auction.status === "live" && Date.now() >= Date.parse(auction.endsAt)) {
        endAuction(auction);
        changed = true;
      }
    }

    if (changed) writeState(state);
    return { state, recovered: loaded.recovered };
  });
}

export async function placeBid({ auctionId, name, email, amount }) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;
    const auction = findAuction(state, auctionId);

    if (!auction) return { ok: false, error: "Lelang tidak ditemukan." };
    if (auction.status !== "live") {
      return { ok: false, error: "Lelang sudah selesai.", state };
    }

    const now = Date.now();
    if (now >= Date.parse(auction.endsAt)) {
      endAuction(auction);
      writeState(state);
      return { ok: false, error: "Lelang sudah selesai.", state };
    }

    if (!name || name.length < 2) {
      return { ok: false, error: "Isi nama dulu sebelum bid.", state };
    }
    if (!email || !email.includes("@")) {
      return { ok: false, error: "Sesi tidak valid. Masuk ulang.", state };
    }
    if (!Number.isSafeInteger(amount) || amount <= 0) {
      return { ok: false, error: "Nominal bid tidak valid.", state };
    }
    if (amount > MAX_AMOUNT) {
      return { ok: false, error: "Nominal bid terlalu besar.", state };
    }
    if (auction.highestBidder === name) {
      return {
        ok: false,
        error: "Kamu sudah memegang harga tertinggi.",
        state,
      };
    }

    const minimum = getMinimum(auction);
    if (amount < minimum) {
      return {
        ok: false,
        error: `Bid minimum ${idr.format(minimum)}.`,
        minimum,
        state,
      };
    }

    const bid = {
      id: nextBidId(auction),
      name,
      email,
      amount,
      at: new Date(now).toISOString(),
    };
    auction.bids.unshift(bid);
    auction.currentPrice = amount;
    auction.highestBidder = name;
    auction.highestBidderEmail = email;

    let extended = false;
    const remaining = Date.parse(auction.endsAt) - now;
    if (remaining <= ANTI_SNIPE_MS && auction.extensionCount < MAX_EXTENSIONS) {
      auction.endsAt = new Date(Date.parse(auction.endsAt) + 60_000).toISOString();
      auction.extensionCount += 1;
      extended = true;
    }

    writeState(state);
    return { ok: true, state, auction, bid, extended };
  });
}

function nextAuctionId(state) {
  return state.auctions.reduce((max, auction) => Math.max(max, auction.id), 0) + 1;
}

function nextLot(state) {
  const maxLot = state.auctions.reduce((max, auction) => {
    const value = Number.parseInt(auction.lot, 10);
    return Number.isFinite(value) ? Math.max(max, value) : max;
  }, 0);
  return String(maxLot + 1).padStart(3, "0");
}

export async function createDraft(input) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;

    const title = String(input.title || "").trim();
    const description = String(input.description || "").trim();
    const condition = String(input.condition || "").trim();
    const startingPrice = Number(input.startingPrice);
    const minIncrement = Number(input.minIncrement);
    const durationMinutes = Number(input.durationMinutes);

    if (title.length < 3) return { ok: false, error: "Judul minimal 3 karakter." };
    if (description.length < 10) {
      return { ok: false, error: "Deskripsi minimal 10 karakter." };
    }
    if (condition.length < 2) return { ok: false, error: "Isi kondisi barang." };
    if (!Number.isSafeInteger(startingPrice) || startingPrice <= 0) {
      return { ok: false, error: "Harga awal harus angka bulat lebih dari 0." };
    }
    if (!Number.isSafeInteger(minIncrement) || minIncrement <= 0) {
      return { ok: false, error: "Kelipatan harus angka bulat lebih dari 0." };
    }
    if (startingPrice > MAX_AMOUNT || minIncrement > MAX_AMOUNT) {
      return { ok: false, error: "Nominal terlalu besar." };
    }
    if (!Number.isFinite(durationMinutes) || durationMinutes < 1 || durationMinutes > 60) {
      return { ok: false, error: "Durasi harus antara 1 dan 60 menit." };
    }

    const auction = makeAuction({
      id: nextAuctionId(state),
      lot: nextLot(state),
      title,
      subtitle: String(input.subtitle || "").trim(),
      description,
      condition,
      includes: String(input.includes || "")
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean),
      specs: String(input.specs || "")
        .split("\n")
        .map((line) => {
          const separator = line.indexOf(":");
          if (separator === -1) return null;
          const label = line.slice(0, separator).trim();
          const value = line.slice(separator + 1).trim();
          if (!label || !value) return null;
          return { label, value };
        })
        .filter(Boolean),
      startingPrice,
      minIncrement,
      durationSeconds: Math.round(durationMinutes * 60),
      status: "draft",
    });

    state.auctions.push(auction);
    writeState(state);
    return { ok: true, state, auction };
  });
}

export async function startAuction(id) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;
    const auction = findAuction(state, id);

    if (!auction) return { ok: false, error: "Lelang tidak ditemukan.", state };
    if (auction.status !== "draft") {
      return { ok: false, error: "Hanya draft yang bisa dimulai.", state };
    }
    if (getLiveAuction(state)) {
      return {
        ok: false,
        error: "Masih ada lelang live. Akhiri lelang itu dulu.",
        state,
      };
    }

    const now = Date.now();
    auction.status = "live";
    auction.startsAt = new Date(now).toISOString();
    auction.endsAt = new Date(now + auction.durationSeconds * 1000).toISOString();
    auction.currentPrice = auction.startingPrice;
    auction.highestBidder = "";
    auction.highestBidderEmail = "";
    auction.winner = "";
    auction.winnerEmail = "";
    auction.finalPrice = null;
    auction.endedAt = null;
    auction.extensionCount = 0;
    auction.bids = [];
    writeState(state);
    return { ok: true, state, auction };
  });
}

export async function endAuctionById(id) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;
    const auction = findAuction(state, id);

    if (!auction) return { ok: false, error: "Lelang tidak ditemukan.", state };
    if (auction.status !== "live") {
      return { ok: false, error: "Lelang ini tidak sedang live.", state };
    }

    endAuction(auction);
    writeState(state);
    return { ok: true, state, auction };
  });
}

export async function deleteDraft(id) {
  return withStateLock(async () => {
    const loaded = readState();
    const state = loaded.state;
    const auction = findAuction(state, id);

    if (!auction) return { ok: false, error: "Lelang tidak ditemukan.", state };
    if (auction.status !== "draft") {
      return { ok: false, error: "Hanya draft yang bisa dihapus.", state };
    }

    state.auctions = state.auctions.filter((item) => item.id !== id);
    writeState(state);
    return { ok: true, state };
  });
}

export async function resetDemo({ withBids = false, withDraft = false } = {}) {
  return withStateLock(async () => {
    const state = seedState({ withBids, withDraft });
    writeState(state);
    return state;
  });
}
