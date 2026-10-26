import { Link, Outlet } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";

export default function AdminLayout() {
  return (
    <>
      <SiteHeader note="Admin prototipe">
        <span className="text-label uppercase text-muted">ADMIN DEMO</span>
        <Link className="text-link" to="/">
          Buka ruang lelang
        </Link>
      </SiteHeader>

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter>
        <p>
          Mode demo admin. Perubahan hanya tersimpan di memori halaman ini dan
          tidak memengaruhi halaman lelang publik.
        </p>
      </SiteFooter>
    </>
  );
}
