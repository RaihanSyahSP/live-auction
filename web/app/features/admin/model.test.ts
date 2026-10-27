import assert from "node:assert/strict";
import test from "node:test";
import type { Auction } from "../auction/types";
import { sortAuctions, toAdminAuction } from "./model.ts";

const baseAuction: Auction = {
  id: "11111111-1111-4111-8111-111111111111",
  created_by: "22222222-2222-4222-8222-222222222222",
  lot: "LOT-ABC12345",
  title: "Kamera uji",
  subtitle: "Body only",
  description: "Kamera untuk pengujian integrasi admin.",
  condition: "Used, 8/10",
  includes: ["Baterai"],
  specs: [{ label: "Sensor", value: "Full-frame" }],
  image_urls: [],
  starting_price: 1_000_000,
  min_increment: 50_000,
  duration_seconds: 600,
  current_price: 1_100_000,
  status: "ended",
  extension_count: 0,
  version: 1,
  winner_id: "33333333-3333-4333-8333-333333333333",
  final_price: 1_100_000,
  created_at: "2026-10-27T10:00:00Z",
};

test("toAdminAuction maps backend fields into admin view model", () => {
  assert.deepEqual(toAdminAuction(baseAuction), {
    id: baseAuction.id,
    lot: baseAuction.lot,
    title: baseAuction.title,
    subtitle: baseAuction.subtitle,
    description: baseAuction.description,
    condition: baseAuction.condition,
    includes: baseAuction.includes,
    specs: baseAuction.specs,
    startingPrice: baseAuction.starting_price,
    minIncrement: baseAuction.min_increment,
    currentPrice: baseAuction.current_price,
    status: baseAuction.status,
    durationSeconds: baseAuction.duration_seconds,
    winnerId: baseAuction.winner_id,
    finalPrice: baseAuction.final_price,
    createdAt: baseAuction.created_at,
  });
});

test("sortAuctions orders live, draft, ended and newest first", () => {
  const live = toAdminAuction({
    ...baseAuction,
    id: "live",
    status: "live",
    winner_id: undefined,
    final_price: undefined,
    created_at: "2026-10-27T09:00:00Z",
  });
  const draftNew = toAdminAuction({
    ...baseAuction,
    id: "draft-new",
    status: "draft",
    winner_id: undefined,
    final_price: undefined,
    created_at: "2026-10-27T12:00:00Z",
  });
  const draftOld = toAdminAuction({
    ...baseAuction,
    id: "draft-old",
    status: "draft",
    winner_id: undefined,
    final_price: undefined,
    created_at: "2026-10-27T08:00:00Z",
  });
  const ended = toAdminAuction({ ...baseAuction, id: "ended" });

  assert.deepEqual(
    sortAuctions([draftOld, ended, live, draftNew]).map((item) => item.id),
    ["live", "draft-new", "draft-old", "ended"],
  );
});
