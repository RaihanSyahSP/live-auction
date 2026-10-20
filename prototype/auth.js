const ACCOUNTS_KEY = "lelang-demo-accounts-v1";
const SESSION_KEY = "lelang-demo-session-v1";

export const DEMO_PASSWORD = "demo123";

const DEMO_ACCOUNTS = [
  { email: "raihan@example.com", name: "Raihan", role: "user" },
  { email: "dewi@example.com", name: "Dewi", role: "user" },
  { email: "admin@example.com", name: "Admin Demo", role: "admin" },
];

let memoryAccounts = null;
let memorySession = null;

// Demo only: passwords stay in localStorage. The backend uses bcrypt and server-side sessions.
function normalizeEmail(value) {
  return String(value || "").trim().toLowerCase();
}

function saveAccounts(accounts) {
  memoryAccounts = accounts;
  try {
    localStorage.setItem(ACCOUNTS_KEY, JSON.stringify(accounts));
  } catch {
    // The in-memory copy keeps the current tab usable.
  }
}

function ensureDemoAccounts(accounts) {
  let changed = false;
  for (const demo of DEMO_ACCOUNTS) {
    if (!accounts.some((account) => account.email === demo.email)) {
      accounts.push({ ...demo, password: DEMO_PASSWORD });
      changed = true;
    }
  }
  if (changed) saveAccounts(accounts);
}

function readAccounts() {
  if (memoryAccounts) return memoryAccounts;

  try {
    const raw = localStorage.getItem(ACCOUNTS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        memoryAccounts = parsed;
        ensureDemoAccounts(memoryAccounts);
        return memoryAccounts;
      }
    }
  } catch {
    // Fall through to a fresh in-memory account list.
  }

  memoryAccounts = [];
  ensureDemoAccounts(memoryAccounts);
  return memoryAccounts;
}

function publicAccount(account) {
  return { email: account.email, name: account.name, role: account.role };
}

function setSession(account) {
  memorySession = publicAccount(account);
  try {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(memorySession));
  } catch {
    // The in-memory session keeps the current tab usable.
  }
}

export function getDemoAccounts() {
  return readAccounts()
    .filter((account) => DEMO_ACCOUNTS.some((demo) => demo.email === account.email))
    .map(publicAccount);
}

export function getSession() {
  if (memorySession) return memorySession;

  try {
    const raw = sessionStorage.getItem(SESSION_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (!parsed?.email) return null;
    const account = readAccounts().find((item) => item.email === parsed.email);
    if (!account) {
      sessionStorage.removeItem(SESSION_KEY);
      return null;
    }
    return publicAccount(account);
  } catch {
    return memorySession;
  }
}

export function login(email, password) {
  const account = readAccounts().find(
    (item) => item.email === normalizeEmail(email),
  );

  if (!account) return { ok: false, error: "Email belum terdaftar." };
  if (account.password !== password) {
    return { ok: false, error: "Password salah." };
  }

  setSession(account);
  return { ok: true, account: getSession() };
}

export function loginDemo(email) {
  const account = readAccounts().find((item) => item.email === email);
  if (!account) return { ok: false, error: "Akun demo tidak ditemukan." };
  setSession(account);
  return { ok: true, account: getSession() };
}

export function register({ name, email, password }) {
  const displayName = String(name || "").trim();
  const normalizedEmail = normalizeEmail(email);

  if (displayName.length < 2) {
    return { ok: false, error: "Nama minimal 2 karakter." };
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
    return { ok: false, error: "Format email belum benar." };
  }
  if (String(password || "").length < 6) {
    return { ok: false, error: "Password minimal 6 karakter." };
  }

  const accounts = readAccounts();
  if (accounts.some((account) => account.email === normalizedEmail)) {
    return { ok: false, error: "Email sudah terdaftar. Coba masuk." };
  }

  const account = {
    email: normalizedEmail,
    name: displayName,
    role: "user",
    password: String(password),
    createdAt: new Date().toISOString(),
  };
  accounts.push(account);
  saveAccounts(accounts);
  setSession(account);
  return { ok: true, account: getSession() };
}

export function logout() {
  memorySession = null;
  try {
    sessionStorage.removeItem(SESSION_KEY);
  } catch {
    // Nothing else to clear when sessionStorage is blocked.
  }
}

export function isAdmin() {
  return getSession()?.role === "admin";
}
