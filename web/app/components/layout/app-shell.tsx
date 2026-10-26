import { Outlet } from "react-router";

export default function AppShell() {
  return (
    <>
      <a
        className="absolute left-2 top-2 z-20 -translate-y-[160%] bg-ink px-3.5 py-2.5 text-sm text-paper focus-visible:translate-y-0 focus-visible:outline-accent"
        href="#main"
      >
        Lewati ke konten
      </a>

      <div className="page flex min-h-dvh flex-col">
        <Outlet />
      </div>
    </>
  );
}
