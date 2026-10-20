import {
  DEMO_PASSWORD,
  getDemoAccounts,
  getSession,
  login,
  loginDemo,
  logout,
} from "./auth.js";
import { registerUrl, safeNext } from "./auth-ui.js";

const next = safeNext(new URLSearchParams(window.location.search).get("next"));

const refs = {
  sessionPanel: document.getElementById("session-panel"),
  sessionName: document.getElementById("session-name"),
  continueLink: document.getElementById("continue-link"),
  logoutCurrent: document.getElementById("logout-current"),
  form: document.getElementById("login-form"),
  email: document.getElementById("email"),
  password: document.getElementById("password"),
  error: document.getElementById("login-error"),
  submit: document.getElementById("login-submit"),
  demoAccounts: document.getElementById("demo-accounts"),
  demoPassword: document.getElementById("demo-password"),
  registerLink: document.getElementById("register-link"),
};

function showError(message) {
  refs.error.textContent = message;
  refs.error.hidden = false;
}

function renderSession() {
  const session = getSession();
  if (!session) {
    refs.sessionPanel.hidden = true;
    refs.form.hidden = false;
    return;
  }

  refs.sessionPanel.hidden = false;
  refs.form.hidden = true;
  refs.sessionName.textContent = session.name;
  refs.continueLink.href = next;
  refs.sessionPanel.dataset.role = session.role;
}

function renderDemoAccounts() {
  const accounts = getDemoAccounts();
  refs.demoAccounts.replaceChildren(
    ...accounts.map((account) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "demo-account";
      button.dataset.demoEmail = account.email;

      const name = document.createElement("span");
      name.textContent = account.name;

      const role = document.createElement("small");
      role.textContent = account.role === "admin" ? "Admin" : "Pembeli";
      button.setAttribute(
        "aria-label",
        `Masuk sebagai ${account.name}, ${role.textContent}`,
      );

      button.append(name, role);
      button.addEventListener("click", () => {
        const result = loginDemo(account.email);
        if (!result.ok) {
          showError(result.error);
          return;
        }
        window.location.href = next;
      });
      return button;
    }),
  );
}

refs.demoPassword.textContent = DEMO_PASSWORD;
refs.registerLink.href = registerUrl(next);
renderDemoAccounts();
renderSession();

refs.form.addEventListener("submit", (event) => {
  event.preventDefault();
  refs.error.hidden = true;

  const email = refs.email.value.trim();
  const password = refs.password.value;
  if (!email || !password) {
    showError("Isi email dan password.");
    return;
  }

  const result = login(email, password);
  if (!result.ok) {
    showError(result.error);
    return;
  }

  window.location.href = next;
});

refs.email.addEventListener("input", () => {
  refs.error.hidden = true;
});

refs.password.addEventListener("input", () => {
  refs.error.hidden = true;
});

refs.logoutCurrent.addEventListener("click", () => {
  logout();
  refs.form.hidden = false;
  renderSession();
  refs.email.focus();
});
