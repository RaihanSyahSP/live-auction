import { Outlet } from "react-router";
import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

export default function PublicLayout() {
  return (
    <>
      <SiteHeader note="Ruang lelang elektronik bekas" />

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter>
        <p>Ruang lelang live untuk elektronik bekas.</p>
      </SiteFooter>
    </>
  );
}
