import assert from "node:assert/strict";
import test from "node:test";
import {
  type DraftFormValues,
  draftFormSchema,
  toDraftInput,
} from "./schema.ts";

const validValues: DraftFormValues = {
  title: "Kamera uji",
  subtitle: "Body only",
  description: "Kamera untuk pengujian form admin.",
  condition: "Used, 8/10",
  includes: "Baterai, Charger",
  specs: "Sensor: Full-frame\nMount: Sony E",
  startingPrice: 1_000_000,
  minIncrement: 50_000,
  durationMinutes: 10,
};

test("draftFormSchema rejects a title shorter than 3 characters", () => {
  const result = draftFormSchema.safeParse({ ...validValues, title: "K" });

  assert.equal(result.success, false);
  if (!result.success) {
    assert.equal(result.error.issues[0]?.message, "Judul minimal 3 karakter.");
  }
});

test("toDraftInput converts includes and specs text into arrays", () => {
  assert.deepEqual(toDraftInput(validValues), {
    title: "Kamera uji",
    subtitle: "Body only",
    description: "Kamera untuk pengujian form admin.",
    condition: "Used, 8/10",
    includes: ["Baterai", "Charger"],
    specs: [
      { label: "Sensor", value: "Full-frame" },
      { label: "Mount", value: "Sony E" },
    ],
    startingPrice: 1_000_000,
    minIncrement: 50_000,
    durationMinutes: 10,
  });
});
