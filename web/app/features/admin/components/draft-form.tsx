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
import { Textarea } from "~/components/ui/textarea";
import { numberFormat } from "~/features/auction/demo";
import type { DraftInput } from "../demo";

type DraftFormProps = {
  onCreate: (input: DraftInput) => void;
};

type Feedback = {
  kind: "error" | "info";
  message: string;
};

type PriceFieldProps = {
  id: string;
  label: string;
  onChange: (value: number) => void;
  placeholder: string;
  value: number;
};

function PriceField({
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
    </Field>
  );
}

export function DraftForm({ onCreate }: DraftFormProps) {
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [startingPrice, setStartingPrice] = useState(0);
  const [minIncrement, setMinIncrement] = useState(0);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const form = event.currentTarget;
    const data = new FormData(form);
    const title = String(data.get("title") ?? "").trim();
    const subtitle = String(data.get("subtitle") ?? "").trim();
    const description = String(data.get("description") ?? "").trim();
    const condition = String(data.get("condition") ?? "").trim();
    const durationMinutes = Number(data.get("durationMinutes"));
    const includes = String(data.get("includes") ?? "")
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
    const specs = String(data.get("specs") ?? "")
      .split("\n")
      .map((line) => {
        const separator = line.indexOf(":");
        if (separator === -1) {
          return null;
        }

        const label = line.slice(0, separator).trim();
        const value = line.slice(separator + 1).trim();
        return label && value ? { label, value } : null;
      })
      .filter((spec): spec is { label: string; value: string } =>
        Boolean(spec),
      );

    if (title.length < 3) {
      setFeedback({ kind: "error", message: "Judul minimal 3 karakter." });
      return;
    }

    if (description.length < 10) {
      setFeedback({ kind: "error", message: "Deskripsi minimal 10 karakter." });
      return;
    }

    if (condition.length < 2) {
      setFeedback({ kind: "error", message: "Isi kondisi barang." });
      return;
    }

    if (startingPrice <= 0) {
      setFeedback({ kind: "error", message: "Harga awal harus lebih dari 0." });
      return;
    }

    if (minIncrement <= 0) {
      setFeedback({ kind: "error", message: "Kelipatan harus lebih dari 0." });
      return;
    }

    if (
      !Number.isFinite(durationMinutes) ||
      durationMinutes < 1 ||
      durationMinutes > 60
    ) {
      setFeedback({
        kind: "error",
        message: "Durasi harus antara 1 dan 60 menit.",
      });
      return;
    }

    onCreate({
      title,
      subtitle,
      description,
      condition,
      includes,
      specs,
      startingPrice,
      minIncrement,
      durationMinutes,
    });

    form.reset();
    setStartingPrice(0);
    setMinIncrement(0);
    setFeedback({
      kind: "info",
      message: "Draft dibuat. Cari di daftar lelang, lalu tekan Start.",
    });
  }

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
        <form id="create-form" noValidate onSubmit={handleSubmit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="auction-title">Judul</FieldLabel>
              <Input
                autoComplete="off"
                id="auction-title"
                maxLength={80}
                name="title"
                placeholder="Misalnya: Fujifilm X-T2"
                required
                type="text"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-subtitle">Subjudul</FieldLabel>
              <Input
                autoComplete="off"
                id="auction-subtitle"
                maxLength={80}
                name="subtitle"
                placeholder="Misalnya: Body only"
                type="text"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-description">Deskripsi</FieldLabel>
              <Textarea
                id="auction-description"
                name="description"
                placeholder="Kondisi, riwayat pemakaian, dan catatan penting."
                required
                rows={4}
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-condition">Kondisi</FieldLabel>
              <Input
                autoComplete="off"
                id="auction-condition"
                maxLength={40}
                name="condition"
                placeholder="Misalnya: Used, 8/10"
                required
                type="text"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-includes">Kelengkapan</FieldLabel>
              <Input
                autoComplete="off"
                id="auction-includes"
                maxLength={160}
                name="includes"
                placeholder="Body, baterai, charger"
                type="text"
              />
              <FieldDescription className="text-[0.78rem]">
                Pisahkan dengan koma.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="auction-specs">Spesifikasi</FieldLabel>
              <Textarea
                id="auction-specs"
                name="specs"
                placeholder={"Sensor: Full-frame 24 MP\nMount: Sony E-mount"}
                rows={3}
              />
              <FieldDescription className="text-[0.78rem]">
                Satu baris per spesifikasi, format Label: Nilai.
              </FieldDescription>
            </Field>

            <div className="grid gap-4 sm:grid-cols-2">
              <PriceField
                id="starting-price"
                label="Harga awal"
                onChange={setStartingPrice}
                placeholder="6.500.000"
                value={startingPrice}
              />
              <PriceField
                id="min-increment"
                label="Kelipatan"
                onChange={setMinIncrement}
                placeholder="100.000"
                value={minIncrement}
              />
              <Field className="sm:col-span-2">
                <FieldLabel htmlFor="duration">Durasi, menit</FieldLabel>
                <Input
                  defaultValue={10}
                  id="duration"
                  inputMode="numeric"
                  max={60}
                  min={1}
                  name="durationMinutes"
                  step={1}
                  type="number"
                />
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

            <Button className="w-full" type="submit">
              Simpan sebagai draft
            </Button>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  );
}
