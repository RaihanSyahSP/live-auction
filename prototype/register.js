import { getSession, logout, register } from "./auth.js";
import { loginUrl, safeNext } from "./auth-ui.js";

const next = safeNext(new URLSearchParams(window.location.search).get("next"));

const refs = {
  sessionPanel: document.getElementById("session-panel"),
  sessionName: document.getElementById("session-name"),
  continueLink: document.getElementById("continue-link"),
  logoutCurrent: document.getElementById("logout-current"),
  form: document.getElementById("register-form"),
  name: document.getElementById("name"),
  email: document.getElementById("email"),
  password: document.getElementById("password"),
  confirmPassword: document.getElementById("confirm-password"),
  error: document.getElementById("register-error"),
  submit: document.getElementById("register-submit"),
  loginLink: document.getElementById("login-link"),
  loginFooter: document.getElementById("login-footer"),
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
}

refs.loginLink.href = loginUrl(next);
refs.loginFooter.href = loginUrl(next);
renderSession();

refs.form.addEventListener("submit", (event) => {
  event.preventDefault();
  refs.error.hidden = true;

  if (refs.password.value !== refs.confirmPassword.value) {
    showError("Password dan ulangi password belum sama.");
    refs.confirmPassword.focus();
    return;
  }

  const result = register({
    name: refs.name.value,
    email: refs.email.value,
    password: refs.password.value,
  });

  if (!result.ok) {
    showError(result.error);
    return;
  }

  window.location.href = next;
});

for (const input of [refs.name, refs.email, refs.password, refs.confirmPassword]) {
  input.addEventListener("input", () => {
    refs.error.hidden = true;
  });
}

refs.logoutCurrent.addEventListener("click", () => {
  logout();
  refs.form.hidden = false;
  renderSession();
  refs.name.focus();
});
