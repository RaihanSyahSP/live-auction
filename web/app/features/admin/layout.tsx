import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { Link, Outlet } from "react-router";
import { SiteFooter } from "~/components/layout/site-footer";
import { SiteHeader } from "~/components/layout/site-header";
import { adminMiddleware, authMiddleware } from "~/features/auth/middleware";

export const middleware = [authMiddleware, adminMiddleware];

export default function AdminLayout() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: 1,
            staleTime: 15_000,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <SiteHeader note="Admin">
        <span className="text-label uppercase text-muted">ADMIN</span>
        <Link className="text-link" to="/">
          Buka ruang lelang
        </Link>
      </SiteHeader>

      <main className="flex-1" id="main">
        <Outlet />
      </main>

      <SiteFooter>
        <p>Perubahan di halaman ini langsung tersimpan ke backend.</p>
      </SiteFooter>
    </QueryClientProvider>
  );
}
