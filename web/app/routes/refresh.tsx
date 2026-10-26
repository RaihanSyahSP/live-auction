import { redirect } from "react-router";
import { safeNext } from "~/features/auth/actions";
import { authApi } from "~/features/auth/api";
import type { Route } from "./+types/refresh";

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const next = safeNext(new URL(request.url).searchParams.get("next"));
  const destination = next.startsWith("/refresh") ? "/" : next;
  const user = await authApi.refresh();

  if (!user) {
    return redirect(`/login?next=${encodeURIComponent(destination)}`);
  }

  return redirect(destination);
}

export default function Refresh() {
  return (
    <main className="page py-10" id="main">
      <p className="text-sm text-muted">Memuat sesi...</p>
    </main>
  );
}
