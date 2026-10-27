import { apiClient, assertOk } from "~/lib/api-client";
import type {
  Auction,
  AuctionDetail,
  AuctionResult,
  PlaceBidResponse,
  Spec,
} from "./types";

export type CreateAuctionInput = {
  title: string;
  subtitle: string;
  description: string;
  condition: string;
  includes: string[];
  specs: Spec[];
  startingPrice: number;
  minIncrement: number;
  durationMinutes: number;
};

function auctionPath(id: string, action?: string) {
  const path = `/admin/auctions/${encodeURIComponent(id)}`;
  return action ? `${path}/${action}` : path;
}

export const auctionApi = {
  async current(): Promise<AuctionDetail | null> {
    const response = await apiClient.fetch("/auctions/current");

    if (response.status === 204) {
      return null;
    }

    await assertOk(response);
    return (await response.json()) as AuctionDetail;
  },

  async detail(id: string): Promise<AuctionDetail> {
    const response = await apiClient.fetch(
      `/auctions/${encodeURIComponent(id)}`,
    );
    await assertOk(response);
    return (await response.json()) as AuctionDetail;
  },

  async results(limit = 20): Promise<AuctionResult[]> {
    const response = await apiClient.fetch(`/results?limit=${limit}`);
    await assertOk(response);
    return (await response.json()) as AuctionResult[];
  },

  async placeBid(
    id: string,
    amount: number,
    idempotencyKey: string,
  ): Promise<PlaceBidResponse> {
    const response = await apiClient.fetch(
      `/auctions/${encodeURIComponent(id)}/bids`,
      {
        method: "POST",
        body: JSON.stringify({
          amount,
          idempotency_key: idempotencyKey,
        }),
      },
    );
    await assertOk(response);
    return (await response.json()) as PlaceBidResponse;
  },

  async listAdmin(page = 1, limit = 50): Promise<Auction[]> {
    const response = await apiClient.fetch(
      `/admin/auctions?page=${page}&limit=${limit}`,
    );
    await assertOk(response);
    return (await response.json()) as Auction[];
  },

  async createAdmin(input: CreateAuctionInput): Promise<Auction> {
    const response = await apiClient.fetch("/admin/auctions", {
      method: "POST",
      body: JSON.stringify({
        title: input.title,
        subtitle: input.subtitle,
        description: input.description,
        condition: input.condition,
        includes: input.includes,
        specs: input.specs,
        starting_price: input.startingPrice,
        min_increment: input.minIncrement,
        duration_minutes: input.durationMinutes,
      }),
    });
    await assertOk(response);
    return (await response.json()) as Auction;
  },

  async startAdmin(id: string): Promise<Auction> {
    const response = await apiClient.fetch(auctionPath(id, "start"), {
      method: "POST",
    });
    await assertOk(response);
    return (await response.json()) as Auction;
  },

  async endAdmin(id: string): Promise<Auction> {
    const response = await apiClient.fetch(auctionPath(id, "end"), {
      method: "POST",
    });
    await assertOk(response);
    return (await response.json()) as Auction;
  },

  async deleteAdmin(id: string): Promise<void> {
    const response = await apiClient.fetch(auctionPath(id), {
      method: "DELETE",
    });
    await assertOk(response);
  },
};
