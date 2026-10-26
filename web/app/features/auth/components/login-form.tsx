import { type FormEvent, useState } from "react";
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

export default function LoginForm() {
  const [feedback, setFeedback] = useState<Feedback | null>(null);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const data = new FormData(event.currentTarget);
    const email = String(data.get("email") ?? "").trim();
    const password = String(data.get("password") ?? "");

    if (!email || !password) {
      setFeedback({ kind: "error", message: "Isi email dan password." });
      return;
    }

    if (!email.includes("@")) {
      setFeedback({ kind: "error", message: "Format email belum valid." });
      return;
    }

    setFeedback({
      kind: "info",
      message: "Form login sudah siap. Integrasi backend belum diaktifkan.",
    });
  }

  return (
    <>
      <section aria-labelledby="login-title">
        <Card>
          <CardHeader>
            <p className="kicker">Akun</p>
            <h1
              className="font-serif text-[clamp(2.3rem,5vw,3.6rem)] leading-none font-normal"
              id="login-title"
            >
              Masuk
            </h1>
            <p className="mt-3 max-w-[52ch] text-[0.95rem] text-muted">
              Gunakan email dan password untuk mengikuti lelang.
            </p>
            <p className="mt-4 border border-rule bg-paper px-3.5 py-2.5 text-sm">
              Mode UI. Login belum terhubung ke backend.
            </p>
          </CardHeader>

          <CardContent>
            <form noValidate onSubmit={handleSubmit}>
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input
                    aria-describedby={feedback ? "login-feedback" : undefined}
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
                    aria-describedby={feedback ? "login-feedback" : undefined}
                    autoComplete="current-password"
                    id="password"
                    name="password"
                    onChange={() => setFeedback(null)}
                    placeholder="Password"
                    required
                    type="password"
                  />

                  <div aria-live="polite">
                    {feedback?.kind === "error" ? (
                      <FieldError id="login-feedback">
                        {feedback.message}
                      </FieldError>
                    ) : null}

                    {feedback?.kind === "info" ? (
                      <FieldDescription
                        className="border border-rule bg-paper px-3 py-2 text-sm text-ink"
                        id="login-feedback"
                        role="status"
                      >
                        {feedback.message}
                      </FieldDescription>
                    ) : null}
                  </div>
                </Field>

                <Field>
                  <Button className="w-full" type="submit">
                    Masuk
                  </Button>
                  <FieldDescription className="mt-3 text-[0.9rem]">
                    Belum punya akun?{" "}
                    <span className="text-ink">
                      Pendaftaran belum tersedia.
                    </span>
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </section>

      <aside aria-labelledby="login-help-title" className="pt-1">
        <p className="kicker">Akun pembeli</p>
        <h2
          className="font-serif text-[1.8rem] leading-[1.15] font-normal"
          id="login-help-title"
        >
          Satu akun untuk mengikuti lelang.
        </h2>
        <ul className="mt-5 grid list-disc gap-2 border-t border-rule-soft pt-4 pl-5 text-sm leading-normal text-muted">
          <li>Pasang bid saat lelang live.</li>
          <li>Lihat status outbid dan riwayat bid.</li>
          <li>Akses akun dan hasil lelang.</li>
        </ul>
      </aside>
    </>
  );
}
