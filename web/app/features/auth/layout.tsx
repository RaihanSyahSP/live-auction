import { Link, Outlet, useMatches } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";
import { type AuthChrome, authChrome } from "~/features/auth/chrome";

export default function AuthLayout() {
  const matches = useMatches();
  const chrome =
    (matches.at(-1)?.handle as { authChrome?: AuthChrome } | undefined)
      ?.authChrome ?? authChrome.login;

  return (
    <>
      <SiteHeader note={chrome.headerNote}>
        <Link className="text-link" to="/">
          Kembali ke ruang
        </Link>
      </SiteHeader>

      <main className="flex-1" id="main">
        <div className="mx-auto grid max-w-[980px] items-start gap-gutter py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
          <Outlet />
        </div>
      </main>

      <SiteFooter
        actions={
          <Link className="text-link" to={chrome.footerAction.to}>
            {chrome.footerAction.label}
          </Link>
        }
      >
        <p>{chrome.footerNote}</p>
      </SiteFooter>
    </>
  );
}
