import { mountAuthLinks } from "./auth-ui.js";
import { connectStateChannel, idr, initState } from "./store.js";

const dateTimeFormat = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
});

const refs = {
  main: document.getElementById("results-main"),
  summary: document.getElementById("results-summary"),
  loading: document.getElementById("results-loading"),
  list: document.getElementById("results-list"),
  empty: document.getElementById("results-empty"),
};

mountAuthLinks(document.getElementById("auth-links"), {
  logoutTarget: "index.html",
});

function renderResults(state) {
  const ended = state.auctions
    .filter((auction) => auction.status === "ended")
    .sort((a, b) => Date.parse(b.endsAt) - Date.parse(a.endsAt));

  refs.summary.textContent = `${ended.length} lelang selesai`;
  refs.loading.hidden = true;
  refs.empty.hidden = ended.length > 0;
  refs.list.replaceChildren(
    ...ended.map((auction) => {
      const article = document.createElement("article");
      article.className = "result-card";

      const top = document.createElement("div");
      top.className = "result-card-top";

      const status = document.createElement("p");
      status.className = "admin-status is-ended";
      status.textContent = "SELESAI";

      const lot = document.createElement("p");
      lot.className = "admin-lot";
      lot.textContent = `LOT ${auction.lot}`;

      top.append(status, lot);

      const title = document.createElement("h2");
      title.className = "result-card-title";
      title.textContent = auction.title;

      article.append(top, title);

      if (auction.subtitle) {
        const subtitle = document.createElement("p");
        subtitle.className = "admin-card-subtitle";
        subtitle.textContent = auction.subtitle;
        article.append(subtitle);
      }

      const meta = document.createElement("dl");
      meta.className = "result-meta";
      addMeta(meta, "Pemenang", auction.winner || "Tidak terjual");
      addMeta(
        meta,
        "Harga akhir",
        typeof auction.finalPrice === "number"
          ? idr.format(auction.finalPrice)
          : "Tidak ada",
      );
      addMeta(meta, "Jumlah bid", `${auction.bids.length} bid`);
      addMeta(
        meta,
        "Selesai",
        dateTimeFormat.format(new Date(auction.endedAt || auction.endsAt)),
      );
      article.append(meta);

      const actions = document.createElement("div");
      actions.className = "admin-actions";
      const link = document.createElement("a");
      link.className = "button button-secondary";
      link.href = `index.html?auction=${auction.id}`;
      link.textContent = "Lihat lot";
      actions.append(link);
      article.append(actions);

      return article;
    }),
  );
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

connectStateChannel({ onState: renderResults });

initState()
  .then(({ state }) => {
    renderResults(state);
    refs.main.setAttribute("aria-busy", "false");
  })
  .catch(() => {
    refs.loading.hidden = true;
    refs.empty.hidden = false;
    refs.empty.textContent = "Hasil lelang tidak bisa dimuat. Muat ulang halaman.";
  });
