import { type FormEvent, useState } from "react";
import { Link } from "react-router";
import { Button } from "~/components/ui/button";
import { Card, CardContent, CardHeader } from "~/components/ui/card";
import {
    Field,
    FieldDescription,
    FieldError,
    FieldGroup,
    FieldLabel,
} from "~/components/ui/field";
import { Input } from "~/components/ui/input";

type Feedback = {
  kind: "error" | "info";
  message: string;
};

export default function RegisterForm() {
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const data = new FormData(event.currentTarget);
    const displayName = String(data.get("display_name") ?? "").trim();
    const email = String(data.get("email") ?? "").trim();
    const password = String(data.get("password") ?? "");
    const confirmPassword = String(data.get("confirmPassword") ?? "");

    if (!displayName || !email || !password || !confirmPassword) {
      setFeedback({ kind: "error", message: "Lengkapi semua field." });
      return;
    }

    if (displayName.length < 3) {
      setFeedback({
        kind: "error",
        message: "Nama tampilan minimal 3 karakter.",
      });
      return;
    }

    if (!email.includes("@")) {
      setFeedback({ kind: "error", message: "Format email belum valid." });
      return;
    }

    if (password.length < 8) {
      setFeedback({
        kind: "error",
        message: "Password minimal 8 karakter.",
      });
      return;
    }

    if (password !== confirmPassword) {
      setFeedback({
        kind: "error",
        message: "Password dan ulangi password belum sama.",
      });
      return;
    }

    setFeedback({
      kind: "info",
      message:
        "Form registrasi sudah siap. Integrasi backend belum diaktifkan.",
    });
  }

  return (
    <>
      <section aria-labelledby="register-title">
        <Card>
          <CardHeader>
            <p className="kicker">Akun baru</p>
            <h1
              className="font-serif text-[clamp(2.3rem,5vw,3.6rem)] leading-none font-normal"
              id="register-title"
            >
              Daftar
            </h1>
            <p className="mt-3 max-w-[52ch] text-[0.95rem] text-muted">
              Akun baru otomatis masuk sebagai pembeli. Kamu bisa langsung bid
              setelah sampai di ruang lelang.
            </p>
            <p className="mt-4 border border-rule bg-paper px-3.5 py-2.5 text-sm">
              Mode UI. Registrasi belum terhubung ke backend.
            </p>
          </CardHeader>

          <CardContent>
            <form noValidate onSubmit={handleSubmit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="name">Nama tampilan</FieldLabel>
                  <Input
                    aria-describedby={
                      feedback ? "register-feedback" : undefined
                    }
                    autoComplete="nickname"
                    id="name"
                    maxLength={50}
                    minLength={3}
                    name="display_name"
                    onChange={() => setFeedback(null)}
                    placeholder="Misalnya: Raihan"
                    required
                    type="text"
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input
                    aria-describedby={
                      feedback ? "register-feedback" : undefined
                    }
                    autoComplete="email"
                    id="email"
                    name="email"
                    onChange={() => setFeedback(null)}
                    placeholder="nama@contoh.com"
                    required
                    type="email"
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <Input
                    aria-describedby={
                      feedback ? "register-feedback" : undefined
                    }
                    autoComplete="new-password"
                    id="password"
                    maxLength={72}
                    minLength={8}
                    name="password"
                    onChange={() => setFeedback(null)}
                    placeholder="Minimal 8 karakter"
                    required
                    type="password"
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="confirm-password">
                    Ulangi password
                  </FieldLabel>
                  <Input
                    aria-describedby={
                      feedback ? "register-feedback" : undefined
                    }
                    autoComplete="new-password"
                    id="confirm-password"
                    maxLength={72}
                    minLength={8}
                    name="confirmPassword"
                    onChange={() => setFeedback(null)}
                    placeholder="Ketik ulang password"
                    required
                    type="password"
                  />
                </Field>

                <FieldDescription className="text-[0.78rem]">
                  Nama tampil di riwayat bid. Email tidak ditampilkan publik.
                </FieldDescription>

                <div aria-live="polite">
                  {feedback?.kind === "error" ? (
                    <FieldError id="register-feedback">
                      {feedback.message}
                    </FieldError>
                  ) : null}

                  {feedback?.kind === "info" ? (
                    <FieldDescription
                      className="border border-rule bg-paper px-3 py-2 text-sm text-ink"
                      id="register-feedback"
                      role="status"
                    >
                      {feedback.message}
                    </FieldDescription>
                  ) : null}
                </div>

                <Field>
                  <Button className="w-full" type="submit">
                    Buat akun
                  </Button>
                  <FieldDescription className="mt-3 text-[0.9rem]">
                    Sudah punya akun?{" "}
                    <Link className="text-ink" to="/login">
                      Masuk
                    </Link>
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </section>

      <aside aria-labelledby="register-help-title" className="pt-1">
        <p className="kicker">Setelah daftar</p>
        <h2
          className="font-serif text-[1.8rem] leading-[1.15] font-normal"
          id="register-help-title"
        >
          Langsung bisa bid
        </h2>
        <p className="mt-4 text-sm leading-normal text-muted">
          Akun baru memakai role pembeli. Admin tidak dibuat lewat halaman ini.
        </p>
        <ul className="mt-5 grid list-disc gap-2 border-t border-rule-soft pt-4 pl-5 text-sm leading-normal text-muted">
          <li>Nama tampil di ledger lelang.</li>
          <li>Email hanya dipakai untuk masuk.</li>
          <li>Bid pertama langsung tersimpan setelah dipasang.</li>
        </ul>
      </aside>
    </>
  );
}
