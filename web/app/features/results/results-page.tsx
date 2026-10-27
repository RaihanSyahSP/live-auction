import { useQuery } from "@tanstack/react-query";
import { Button } from "~/components/ui/button";
import { auctionApi } from "~/features/auction/api";
import { getErrorMessage } from "~/lib/api-client";
import { ResultCard } from "./components/result-card";

const resultsKey = ["results"] as const;

export default function ResultsPage() {
  const resultsQuery = useQuery({
    queryKey: resultsKey,
    queryFn: () => auctionApi.results(),
    enabled: typeof window !== "undefined",
  });

  const results = resultsQuery.data ?? [];

  return (
    <section
      aria-labelledby="results-title"
      className="py-[clamp(28px,5vw,56px)]"
    >
      <div className="border-b-2 border-ink pb-3">
        <p className="kicker">Arsip</p>
        <h1
          className="font-serif text-[clamp(2.3rem,5vw,3.6rem)] font-normal leading-none"
          id="results-title"
        >
          Hasil lelang
        </h1>
        <p className="mt-2 text-[0.82rem] text-muted tabular-nums">
          {resultsQuery.isSuccess
            ? `${results.length} lelang selesai`
            : "Memuat arsip..."}
        </p>
      </div>

      {resultsQuery.isPending ? (
        <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
          Memuat hasil lelang...
        </p>
      ) : null}

      {resultsQuery.isError ? (
        <div
          className="mt-4 border border-accent bg-paper px-3.5 py-3 text-sm"
          role="alert"
        >
          <p className="font-semibold">Gagal memuat hasil lelang.</p>
          <p className="mt-1 text-muted">
            {getErrorMessage(resultsQuery.error)}
          </p>
          <Button
            className="mt-3"
            onClick={() => void resultsQuery.refetch()}
            size="sm"
            type="button"
            variant="secondary"
          >
            Coba lagi
          </Button>
        </div>
      ) : null}

      {resultsQuery.isSuccess ? (
        results.length > 0 ? (
          <div className="mt-4 grid gap-3">
            {results.map((result) => (
              <ResultCard key={result.auction.id} result={result} />
            ))}
          </div>
        ) : (
          <div className="mt-4 border border-rule bg-surface p-5 text-[0.9rem] text-muted">
            <p>Belum ada lelang yang selesai.</p>
            <a className="text-link mt-2" href="/">
              Buka ruang lelang
            </a>
          </div>
        )
      ) : null}
    </section>
  );
}
