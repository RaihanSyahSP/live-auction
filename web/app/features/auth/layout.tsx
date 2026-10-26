import { Link, Outlet } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";

export default function AuthLayout() {
  return (
    <>
      <SiteHeader note="Masuk ke ruang lelang">
        <Link className="text-link" to="/">
          Kembali ke ruang
        </Link>
      </SiteHeader>

      <main className="flex-1" id="main">
        <div className="mx-auto grid max-w-[980px] items-start gap-gutter py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
          <Outlet />
        </div>
      </main>

      <SiteFooter>
        <p>Masuk untuk memasang bid dan mengikuti lelang.</p>
      </SiteFooter>
    </>
  );
}
