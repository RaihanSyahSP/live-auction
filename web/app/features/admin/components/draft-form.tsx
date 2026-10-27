import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
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
import { Textarea } from "~/components/ui/textarea";
import { numberFormat } from "~/features/auction/demo";
import { getErrorMessage } from "~/lib/api-client";
import type { DraftInput } from "../model";
import { type DraftFormValues, draftResolver, toDraftInput } from "../schema";

type DraftFormProps = {
  onCreate: (input: DraftInput) => Promise<void>;
};

type Feedback = {
  kind: "error" | "info";
  message: string;
};

type PriceFieldProps = {
  error?: string;
  id: string;
  label: string;
  onChange: (value: number) => void;
  placeholder: string;
  value: number;
};

function PriceField({
  error,
  id,
  label,
  onChange,
  placeholder,
  value,
}: PriceFieldProps) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <div className="flex items-center rounded-xs border-[1.5px] border-rule bg-white focus-within:border-ink focus-within:[outline:3px_solid_var(--color-ink)] focus-within:outline-offset-2">
        <span className="pl-3 text-[0.9rem] text-muted">Rp</span>
        <Input
          aria-describedby={error ? `${id}-error` : undefined}
          aria-invalid={Boolean(error)}
          autoComplete="off"
          className="min-h-11 min-w-0 flex-1 rounded-none border-0 bg-transparent px-2 py-2.5 tabular-nums focus-visible:outline-none"
          id={id}
          inputMode="numeric"
          onChange={(event) => {
            const digits = event.target.value.replace(/\D/g, "").slice(0, 13);
            onChange(digits ? Number(digits) : 0);
          }}
          placeholder={placeholder}
          type="text"
          value={value > 0 ? numberFormat.format(value) : ""}
        />
      </div>
      {error ? <FieldError id={`${id}-error`}>{error}</FieldError> : null}
    </Field>
  );
}

export function DraftForm({ onCreate }: DraftFormProps) {
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
    reset,
  } = useForm<DraftFormValues>({
    defaultValues: {
      condition: "",
      description: "",
      durationMinutes: 10,
      includes: "",
      minIncrement: 0,
      specs: "",
      startingPrice: 0,
      subtitle: "",
      title: "",
    },
    resolver: draftResolver,
  });

  const submit = handleSubmit(async (values) => {
    setFeedback(null);

    try {
      await onCreate(toDraftInput(values));
      reset();
      setFeedback({
        kind: "info",
        message: "Draft dibuat. Cari di daftar lelang, lalu tekan Start.",
      });
    } catch (error) {
      setFeedback({ kind: "error", message: getErrorMessage(error) });
    }
  });

  return (
    <Card>
      <CardHeader>
        <p className="kicker">Lelang baru</p>
        <h2
          className="font-serif text-[2rem] font-normal leading-tight"
          id="draft-title"
        >
          Buat draft
        </h2>
        <FieldDescription className="mt-1">
          Foto memakai ilustrasi placeholder. Tidak ada upload file pada tahap
          ini.
        </FieldDescription>
      </CardHeader>

      <CardContent>
        <form id="create-form" noValidate onSubmit={submit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="auction-title">Judul</FieldLabel>
              <Input
                aria-describedby={
                  errors.title ? "auction-title-error" : undefined
                }
                aria-invalid={Boolean(errors.title)}
                autoComplete="off"
                id="auction-title"
                maxLength={120}
                placeholder="Misalnya: Fujifilm X-T2"
                type="text"
                {...register("title")}
              />
              {errors.title ? (
                <FieldError id="auction-title-error">
                  {errors.title.message}
                </FieldError>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-subtitle">Subjudul</FieldLabel>
              <Input
                aria-describedby={
                  errors.subtitle ? "auction-subtitle-error" : undefined
                }
                aria-invalid={Boolean(errors.subtitle)}
                autoComplete="off"
                id="auction-subtitle"
                maxLength={120}
                placeholder="Misalnya: Body only"
                type="text"
                {...register("subtitle")}
              />
              {errors.subtitle ? (
                <FieldError id="auction-subtitle-error">
                  {errors.subtitle.message}
                </FieldError>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-description">Deskripsi</FieldLabel>
              <Textarea
                aria-describedby={
                  errors.description ? "auction-description-error" : undefined
                }
                aria-invalid={Boolean(errors.description)}
                id="auction-description"
                placeholder="Kondisi, riwayat pemakaian, dan catatan penting."
                rows={4}
                {...register("description")}
              />
              {errors.description ? (
                <FieldError id="auction-description-error">
                  {errors.description.message}
                </FieldError>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-condition">Kondisi</FieldLabel>
              <Input
                aria-describedby={
                  errors.condition ? "auction-condition-error" : undefined
                }
                aria-invalid={Boolean(errors.condition)}
                autoComplete="off"
                id="auction-condition"
                maxLength={40}
                placeholder="Misalnya: Used, 8/10"
                type="text"
                {...register("condition")}
              />
              {errors.condition ? (
                <FieldError id="auction-condition-error">
                  {errors.condition.message}
                </FieldError>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-includes">Kelengkapan</FieldLabel>
              <Input
                aria-describedby={
                  errors.includes ? "auction-includes-error" : undefined
                }
                aria-invalid={Boolean(errors.includes)}
                autoComplete="off"
                id="auction-includes"
                maxLength={160}
                placeholder="Body, baterai, charger"
                type="text"
                {...register("includes")}
              />
              <FieldDescription className="text-[0.78rem]">
                Pisahkan dengan koma.
              </FieldDescription>
              {errors.includes ? (
                <FieldError id="auction-includes-error">
                  {errors.includes.message}
                </FieldError>
              ) : null}
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-specs">Spesifikasi</FieldLabel>
              <Textarea
                aria-describedby={
                  errors.specs ? "auction-specs-error" : undefined
                }
                aria-invalid={Boolean(errors.specs)}
                id="auction-specs"
                placeholder={"Sensor: Full-frame 24 MP\nMount: Sony E-mount"}
                rows={3}
                {...register("specs")}
              />
              <FieldDescription className="text-[0.78rem]">
                Satu baris per spesifikasi, format Label: Nilai.
              </FieldDescription>
              {errors.specs ? (
                <FieldError id="auction-specs-error">
                  {errors.specs.message}
                </FieldError>
              ) : null}
            </Field>

            <div className="grid gap-4 sm:grid-cols-2">
              <Controller
                control={control}
                name="startingPrice"
                render={({ field }) => (
                  <PriceField
                    error={errors.startingPrice?.message}
                    id="starting-price"
                    label="Harga awal"
                    onChange={field.onChange}
                    placeholder="6.500.000"
                    value={field.value}
                  />
                )}
              />
              <Controller
                control={control}
                name="minIncrement"
                render={({ field }) => (
                  <PriceField
                    error={errors.minIncrement?.message}
                    id="min-increment"
                    label="Kelipatan"
                    onChange={field.onChange}
                    placeholder="100.000"
                    value={field.value}
                  />
                )}
              />
              <Field className="sm:col-span-2">
                <FieldLabel htmlFor="duration">Durasi, menit</FieldLabel>
                <Input
                  aria-describedby={
                    errors.durationMinutes ? "duration-error" : undefined
                  }
                  aria-invalid={Boolean(errors.durationMinutes)}
                  id="duration"
                  inputMode="numeric"
                  max={60}
                  min={1}
                  step={1}
                  type="number"
                  {...register("durationMinutes", { valueAsNumber: true })}
                />
                {errors.durationMinutes ? (
                  <FieldError id="duration-error">
                    {errors.durationMinutes.message}
                  </FieldError>
                ) : null}
              </Field>
            </div>

            <div aria-live="polite">
              {feedback?.kind === "error" ? (
                <FieldError>{feedback.message}</FieldError>
              ) : null}
              {feedback?.kind === "info" ? (
                <FieldDescription
                  className="border border-rule bg-paper px-3 py-2 text-ink"
                  role="status"
                >
                  {feedback.message}
                </FieldDescription>
              ) : null}
            </div>

            <Button className="w-full" disabled={isSubmitting} type="submit">
              {isSubmitting ? "Menyimpan..." : "Simpan sebagai draft"}
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  );
}
