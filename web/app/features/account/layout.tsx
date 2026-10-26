import { Link, Outlet } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";

export default function AccountLayout() {
  return (
    <>
      <SiteHeader note="Akun pembeli">
        <span className="text-label uppercase text-muted">AKUN DEMO</span>
        <Link className="text-link" to="/">
          Buka ruang lelang
        </Link>
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
