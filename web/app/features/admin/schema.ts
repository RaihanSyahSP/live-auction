import type { FieldErrors, Resolver } from "react-hook-form";
import { z } from "zod";
import type { DraftInput } from "./model";

export const draftFormSchema = z.object({
  title: z
    .string()
    .trim()
    .min(3, "Judul minimal 3 karakter.")
    .max(120, "Judul maksimal 120 karakter."),
  subtitle: z.string().trim().max(120, "Subjudul maksimal 120 karakter."),
  description: z.string().trim().min(10, "Deskripsi minimal 10 karakter."),
  condition: z.string().trim().min(2, "Isi kondisi barang."),
  includes: z.string().trim(),
  specs: z.string().trim(),
  startingPrice: z
    .number()
    .int("Harga awal harus bilangan bulat.")
    .positive("Harga awal harus lebih dari 0."),
  minIncrement: z
    .number()
    .int("Kelipatan harus bilangan bulat.")
    .positive("Kelipatan harus lebih dari 0."),
  durationMinutes: z
    .number({ error: "Durasi harus antara 1 dan 60 menit." })
    .int("Durasi harus bilangan bulat.")
    .min(1, "Durasi harus antara 1 dan 60 menit.")
    .max(60, "Durasi harus antara 1 dan 60 menit."),
});

export type DraftFormValues = z.infer<typeof draftFormSchema>;

export const draftResolver: Resolver<DraftFormValues> = async (values) => {
  const result = draftFormSchema.safeParse(values);

  if (result.success) {
    return { values: result.data, errors: {} };
  }

  const errors: FieldErrors<DraftFormValues> = {};

  for (const issue of result.error.issues) {
    const field = issue.path[0];
    if (typeof field === "string" && !errors[field as keyof DraftFormValues]) {
      errors[field as keyof DraftFormValues] = {
        message: issue.message,
        type: issue.code,
      };
    }
  }

  return { values: {}, errors };
};

export function toDraftInput(values: DraftFormValues): DraftInput {
  const includes = values.includes
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);

  const specs = values.specs
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
    .filter((spec): spec is { label: string; value: string } => Boolean(spec));

  return {
    title: values.title,
    subtitle: values.subtitle,
    description: values.description,
    condition: values.condition,
    includes,
    specs,
    startingPrice: values.startingPrice,
    minIncrement: values.minIncrement,
    durationMinutes: values.durationMinutes,
  };
}
