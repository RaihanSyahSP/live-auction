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

type RegisterFormProps = {
  error?: string;
  errors?: Record<string, string[] | undefined>;
  fields?: { displayName?: string; email?: string };
  isSubmitting?: boolean;
};

export default function RegisterForm({
  error,
  errors,
  fields,
  isSubmitting = false,
}: RegisterFormProps) {
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
          </CardHeader>

          <CardContent>
            <Form method="post" noValidate>
              <FieldGroup>
                {error ? <FieldError>{error}</FieldError> : null}

                <Field>
                  <FieldLabel htmlFor="name">Nama tampilan</FieldLabel>
                  <Input
                    aria-describedby={
                      errors?.displayName ? "name-error" : undefined
                    }
                    aria-invalid={Boolean(errors?.displayName)}
                    autoComplete="nickname"
                    defaultValue={fields?.displayName}
                    id="name"
                    maxLength={50}
                    minLength={3}
                    name="display_name"
                    placeholder="Misalnya: Raihan"
                    required
                    type="text"
                  />
                  {errors?.displayName ? (
                    <FieldError id="name-error">
                      {errors.displayName[0]}
                    </FieldError>
                  ) : null}
                </Field>

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
                    autoComplete="new-password"
                    id="password"
                    maxLength={72}
                    minLength={8}
                    name="password"
                    placeholder="Minimal 8 karakter"
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
                  <FieldLabel htmlFor="confirm-password">
                    Ulangi password
                  </FieldLabel>
                  <Input
                    aria-describedby={
                      errors?.confirmPassword
                        ? "confirm-password-error"
                        : undefined
                    }
                    aria-invalid={Boolean(errors?.confirmPassword)}
                    autoComplete="new-password"
                    id="confirm-password"
                    maxLength={72}
                    minLength={8}
                    name="confirmPassword"
                    placeholder="Ketik ulang password"
                    required
                    type="password"
                  />
                  {errors?.confirmPassword ? (
                    <FieldError id="confirm-password-error">
                      {errors.confirmPassword[0]}
                    </FieldError>
                  ) : null}
                </Field>

                <FieldDescription className="text-[0.78rem]">
                  Nama tampil di riwayat bid. Email tidak ditampilkan publik.
                </FieldDescription>

                <Field>
                  <Button
                    className="w-full"
                    disabled={isSubmitting}
                    type="submit"
                  >
                    {isSubmitting ? "Memproses..." : "Buat akun"}
                  </Button>
                  <FieldDescription className="mt-3 text-[0.9rem]">
                    Sudah punya akun?{" "}
                    <Link className="text-ink" to="/login">
                      Masuk
                    </Link>
                  </FieldDescription>
                </Field>
              </FieldGroup>
            </Form>
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
