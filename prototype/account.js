import { getSession, logout } from "./auth.js";
import { loginUrl, mountAuthLinks } from "./auth-ui.js";
import {
  connectStateChannel,
  idr,
  initState,
  timeFormat,
} from "./store.js";

const session = getSession();

if (!session) {
  window.location.replace(loginUrl("account.html"));
} else {
  mountAuthLinks(document.getElementById("auth-links"), {
    logoutTarget: "index.html",
  });

  const refs = {
    main: document.getElementById("account-main"),
    profile: document.getElementById("account-profile"),
    name: document.getElementById("profile-name"),
    email: document.getElementById("profile-email"),
    role: document.getElementById("profile-role"),
    adminLink: document.getElementById("admin-link"),
    logout: document.getElementById("logout-account"),
    count: document.getElementById("my-bid-count"),
    loading: document.getElementById("account-loading"),
    list: document.getElementById("account-bid-list"),
    empty: document.getElementById("account-empty"),
  };

  function renderBids(state) {
    const rows = [];

    for (const auction of state.auctions) {
      for (const bid of auction.bids) {
        if (bid.email !== session.email) continue;

        let label = "Draft";
        let statusClass = "is-outbid";

        if (auction.status === "live") {
          if (auction.highestBidderEmail === session.email) {
            label = "Tertinggi";
            statusClass = "is-leading";
          } else {
            label = "Terlampaui";
          }
        } else if (auction.status === "ended") {
          if (auction.winnerEmail === session.email) {
            label = "Menang";
            statusClass = "is-won";
          } else {
            label = "Tidak menang";
          }
        }

        rows.push({ auction, bid, label, statusClass });
      }
    }

    rows.sort((a, b) => Date.parse(b.bid.at) - Date.parse(a.bid.at));

    refs.count.textContent = `${rows.length} bid`;
    refs.loading.hidden = true;
    refs.empty.hidden = rows.length > 0;
    refs.list.replaceChildren(
      ...rows.map(({ auction, bid, label, statusClass }) => {
        const item = document.createElement("li");
        item.className = "account-bid";

        const main = document.createElement("div");
        main.className = "account-bid-main";

        const title = document.createElement("p");
        title.className = "account-bid-title";
        const link = document.createElement("a");
        link.href = `index.html?auction=${auction.id}`;
        link.textContent = auction.title;
        title.append(link);

        const meta = document.createElement("p");
        meta.className = "account-bid-meta";
        meta.textContent = `LOT ${auction.lot} · ${timeFormat.format(
          new Date(bid.at),
        )} · ${idr.format(bid.amount)}`;

        main.append(title, meta);

        const status = document.createElement("p");
        status.className = `bid-state ${statusClass}`;
        status.textContent = label;

        item.append(main, status);
        return item;
      }),
    );
  }

  refs.name.textContent = session.name;
  refs.email.textContent = session.email;
  refs.role.textContent = session.role === "admin" ? "Admin" : "Pembeli";
  refs.adminLink.hidden = session.role !== "admin";
  refs.profile.hidden = false;
  refs.main.setAttribute("aria-busy", "false");

  refs.logout.addEventListener("click", () => {
    logout();
    window.location.href = "index.html";
  });

  connectStateChannel({
    onState: (state) => {
      renderBids(state);
    },
  });

  initState()
    .then(({ state }) => {
      renderBids(state);
    })
    .catch(() => {
      refs.loading.hidden = true;
      refs.empty.hidden = false;
      refs.empty.textContent = "Bid tidak bisa dimuat. Muat ulang halaman.";
    });
}
