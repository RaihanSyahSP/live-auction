import { getSession, logout } from "./auth.js";

export function safeNext(raw, fallback = "index.html") {
  if (!raw) return fallback;
  try {
    const url = new URL(raw, window.location.origin);
    if (url.origin !== window.location.origin) return fallback;
    if (url.pathname !== "/" && !url.pathname.endsWith(".html")) return fallback;
    return `${url.pathname.replace(/^\//, "")}${url.search}`;
  } catch {
    return fallback;
  }
}

export function currentPath() {
  return `${window.location.pathname.replace(/^\//, "")}${window.location.search}`;
}

export function loginUrl(next = currentPath()) {
  const params = new URLSearchParams({ next });
  return `login.html?${params.toString()}`;
}

export function registerUrl(next = currentPath()) {
  const params = new URLSearchParams({ next });
  return `register.html?${params.toString()}`;
}

export function mountAuthLinks(container, { logoutTarget = "index.html" } = {}) {
  if (!container) return;
  container.replaceChildren();
  const session = getSession();

  if (session) {
    const account = document.createElement("a");
    account.className = "text-link";
    account.href = "account.html";
    account.textContent = `Akun: ${session.name}`;

    const button = document.createElement("button");
    button.type = "button";
    button.className = "text-button";
    button.textContent = "Keluar";
    button.addEventListener("click", () => {
      logout();
      window.location.href = logoutTarget;
    });

    container.append(account, button);
    return;
  }

  const next = currentPath();
  const login = document.createElement("a");
  login.className = "text-link";
  login.href = loginUrl(next);
  login.textContent = "Masuk";

  const register = document.createElement("a");
  register.className = "text-link";
  register.href = registerUrl(next);
  register.textContent = "Daftar";

  container.append(login, register);
}
