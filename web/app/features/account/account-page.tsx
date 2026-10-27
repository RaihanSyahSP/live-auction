import { Form, Link, useOutletContext } from "react-router";
import { Button } from "~/components/ui/button";
import { Card } from "~/components/ui/card";
import { idr } from "~/features/auction/format";
import type { User } from "../auth/types";
import { type AccountBidStatus, bidStatusLabel, demoAccountBids } from "./demo";

const secondaryLinkClass =
  "inline-flex min-h-11 items-center justify-center rounded-xs border border-ink px-3 py-2 text-sm font-medium text-ink hover:bg-ink hover:text-paper";

function statusClass(status: AccountBidStatus) {
  if (status === "leading") {
    return "text-accent";
  }
  if (status === "won") {
    return "text-ink";
  }
  return "text-muted";
}

export default function AccountPage() {
  const { user } = useOutletContext<{ user: User | null }>();

  if (!user) {
    return null;
  }

  const name = user.display_name ?? user.email;
  const email = user.email;
  const role = user.role === "admin" ? "Admin" : "Pembeli";

  return (
    <>
      <p className="mt-4 border border-rule bg-surface px-3.5 py-3 text-sm">
        <strong>Data demo.</strong> Profil dan riwayat bid di halaman ini belum
        terhubung ke backend.
      </p>

      <div className="grid items-start gap-gutter py-[clamp(28px,5vw,56px)] lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <section aria-labelledby="profile-title">
          <Card className="p-[clamp(20px,3vw,32px)]">
            <p className="kicker">Akun</p>
            <h1
              className="font-serif text-[clamp(2.3rem,5vw,3.6rem)] font-normal leading-none"
              id="profile-title"
            >
              Profil
            </h1>

            <dl className="mt-5 border-t border-rule-soft">
              <div className="grid grid-cols-[80px_minmax(0,1fr)] gap-3 border-b border-rule-soft py-2.5">
                <dt className="text-label uppercase text-muted">Nama</dt>
                <dd className="break-words text-[0.9rem]">{name}</dd>
              </div>
              <div className="grid grid-cols-[80px_minmax(0,1fr)] gap-3 border-b border-rule-soft py-2.5">
                <dt className="text-label uppercase text-muted">Email</dt>
                <dd className="break-words text-[0.9rem]">{email}</dd>
              </div>
              <div className="grid grid-cols-[80px_minmax(0,1fr)] gap-3 border-b border-rule-soft py-2.5">
                <dt className="text-label uppercase text-muted">Peran</dt>
                <dd className="text-[0.9rem]">{role}</dd>
              </div>
            </dl>

            <div className="mt-5 flex flex-wrap gap-2">
              <Link className={secondaryLinkClass} to="/">
                Buka ruang
              </Link>
              <Form method="post" action="/logout">
                <Button size="sm" type="submit" variant="secondary">
                  Keluar
                </Button>
              </Form>
            </div>
          </Card>
        </section>

        <section aria-labelledby="my-bids-title">
          <div className="flex items-baseline justify-between gap-4 border-b-2 border-ink pb-2.5">
            <div>
              <p className="kicker">Aktivitas</p>
              <h2
                className="font-serif text-[1.7rem] font-normal leading-normal"
                id="my-bids-title"
              >
                Bid saya
              </h2>
            </div>
            <p className="text-[0.8rem] text-muted tabular-nums">
              {demoAccountBids.length} bid
            </p>
          </div>

          {demoAccountBids.length > 0 ? (
            <ol>
              {demoAccountBids.map((bid) => (
                <li
                  className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 border-b border-rule-soft py-3.5"
                  key={bid.id}
                >
                  <div>
                    <p className="text-[0.95rem] font-semibold">{bid.title}</p>
                    <p className="mt-1 text-[0.78rem] text-muted tabular-nums">
                      LOT {bid.lot} · {bid.at} · {idr.format(bid.amount)}
                    </p>
                  </div>
                  <p
                    className={`text-[0.68rem] font-bold tracking-[0.06em] uppercase ${statusClass(bid.status)}`}
                  >
                    {bidStatusLabel[bid.status]}
                  </p>
                </li>
              ))}
            </ol>
          ) : (
            <p className="border-b border-rule-soft py-[18px] text-[0.9rem] text-muted">
              Belum ada bid. Buka ruang lelang dan pasang bid pertama.
            </p>
          )}

          <p className="mt-3 text-[0.75rem] text-muted">
            Urutan terbaru di atas. Waktu memakai jam perangkat ini.
          </p>
        </section>
      </div>
    </>
  );
}
