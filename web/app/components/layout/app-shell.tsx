import { Outlet, useLoaderData } from "react-router";
import { userContext } from "~/context/auth";
import { sessionMiddleware } from "~/features/auth/middleware";
import type { Route } from "./+types/app-shell";

export const middleware = [sessionMiddleware];

export async function loader({ context }: Route.LoaderArgs) {
  return {
    user: context.get(userContext),
  };
}

export default function AppShell() {
  const { user } = useLoaderData<typeof loader>();

  return (
    <>
      <a
        className="absolute left-2 top-2 z-20 -translate-y-[160%] bg-ink px-3.5 py-2.5 text-sm text-paper focus-visible:translate-y-0 focus-visible:outline-accent"
        href="#main"
      >
        Lewati ke konten
      </a>

      <div className="page flex min-h-dvh flex-col">
        <Outlet context={{ user }} />
      </div>
    </>
  );
}
