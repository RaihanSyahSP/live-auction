import { Link, Outlet, useMatches } from "react-router";
import type { PublicChrome } from "./public-chrome";
import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

export default function PublicLayout() {
  const matches = useMatches();
  const chrome = (
    matches.at(-1)?.handle as { publicChrome?: PublicChrome } | undefined
  )?.publicChrome ?? {
    note: "Ruang lelang elektronik bekas",
  };

  return (
    <>
      <SiteHeader note={chrome.note}>
        {chrome.lotLabel ? (
          <span className="text-label uppercase text-muted">
            {chrome.lotLabel}
          </span>
        ) : null}
        {chrome.demoTag ? (
          <span className="text-label uppercase text-muted">
            {chrome.demoTag}
          </span>
        ) : null}
        <Link className="text-link" to="/login">
          Masuk
        </Link>
        <Link className="text-link" to="/register">
          Daftar
        </Link>
      </SiteHeader>

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter>
        <p>
          {chrome.footerNote ?? "Ruang lelang live untuk elektronik bekas."}
        </p>
      </SiteFooter>
    </>
  );
}
