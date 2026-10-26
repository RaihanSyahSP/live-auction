import { Form, Link } from "react-router";
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

type LoginFormProps = {
  error?: string;
  errors?: Record<string, string[] | undefined>;
  fields?: { email?: string };
  isSubmitting?: boolean;
};

export default function LoginForm({
  error,
  errors,
  fields,
  isSubmitting = false,
}: LoginFormProps) {
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
          </CardHeader>

          <CardContent>
            <Form method="post" noValidate>
              <FieldGroup>
                {error ? <FieldError>{error}</FieldError> : null}

                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input
                    aria-describedby={errors?.email ? "email-error" : undefined}
                    aria-invalid={Boolean(errors?.email)}
                    autoComplete="email"
                    defaultValue={fields?.email}
                    id="email"
                    name="email"
                    placeholder="nama@contoh.com"
                    required
                    type="email"
                  />
                  {errors?.email ? (
                    <FieldError id="email-error">{errors.email[0]}</FieldError>
                  ) : null}
                </Field>

                <Field>
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <Input
                    aria-describedby={
                      errors?.password ? "password-error" : undefined
                    }
                    aria-invalid={Boolean(errors?.password)}
                    autoComplete="current-password"
                    id="password"
                    name="password"
                    placeholder="Password"
                    required
                    type="password"
                  />
                  {errors?.password ? (
                    <FieldError id="password-error">
                      {errors.password[0]}
                    </FieldError>
                  ) : null}
                </Field>

                <Field>
                  <Button
                    className="w-full"
                    disabled={isSubmitting}
                    type="submit"
                  >
                    {isSubmitting ? "Memproses..." : "Masuk"}
                  </Button>
                  <FieldDescription className="mt-3 text-[0.9rem]">
                    Belum punya akun?{" "}
                    <Link className="text-ink" to="/register">
                      Daftar
                    </Link>
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </Form>
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
