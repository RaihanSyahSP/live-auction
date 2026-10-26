import { ResultCard } from "./components/result-card";
import { demoResults } from "./demo";

export default function ResultsPage() {
  return (
    <>
      <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
        <strong>Data demo.</strong> Hasil lelang di halaman ini belum diambil
        dari backend.
      </p>

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
            {demoResults.length} lelang selesai
          </p>
        </div>

        {demoResults.length > 0 ? (
          <div className="mt-4 grid gap-3">
            {demoResults.map((result) => (
              <ResultCard key={result.id} result={result} />
            ))}
          </div>
        ) : (
          <div className="mt-4 border border-rule bg-surface p-5 text-[0.9rem] text-muted">
            <p>Belum ada lelang yang selesai.</p>
            <a className="text-link mt-2" href="/">
              Buka ruang lelang
            </a>
          </div>
        )}
      </section>
    </>
  );
}
