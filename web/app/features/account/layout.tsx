import { Link, Outlet, useOutletContext } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";
import { authMiddleware } from "~/features/auth/middleware";
import type { User } from "~/features/auth/types";

export const middleware = [authMiddleware];

export default function AccountLayout() {
  const { user } = useOutletContext<{ user: User | null }>();

  return (
    <>
      <SiteHeader note="Akun pembeli">
        <span className="text-label uppercase text-muted">AKUN DEMO</span>
        <Link className="text-link" to="/">
          Buka ruang lelang
        </Link>
        {user ? (
          <span className="text-label uppercase text-muted">
            {user.display_name ?? user.email}
          </span>
        ) : null}
      </SiteHeader>

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter
        actions={
          <Link className="text-link" to="/results">
            Lihat hasil lelang
          </Link>
        }
      >
        <p>Data demo. Profil dan bid belum berasal dari backend.</p>
      </SiteFooter>
    </>
  );
}
