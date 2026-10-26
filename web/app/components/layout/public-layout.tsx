import { Link, Outlet, useMatches, useOutletContext } from "react-router";
import type { User } from "~/features/auth/types";
import type { PublicChrome } from "./public-chrome";
import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

export default function PublicLayout() {
  const { user } = useOutletContext<{ user: User | null }>();
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
        {user ? (
          <Link className="text-link" to="/account">
            Akun: {user.display_name ?? user.email}
          </Link>
        ) : (
          <>
            <Link className="text-link" to="/login">
              Masuk
            </Link>
            <Link className="text-link" to="/register">
              Daftar
            </Link>
          </>
        )}
      </SiteHeader>

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter
        actions={
          chrome.footerActions?.length ? (
            <div className="flex flex-wrap gap-x-4 gap-y-1">
              {chrome.footerActions.map((action) => (
                <Link className="text-link" key={action.to} to={action.to}>
                  {action.label}
                </Link>
              ))}
            </div>
          ) : undefined
        }
      >
        <p>
          {chrome.footerNote ?? "Ruang lelang live untuk elektronik bekas."}
        </p>
      </SiteFooter>
    </>
  );
}
