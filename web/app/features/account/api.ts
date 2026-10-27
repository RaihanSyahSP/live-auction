import { apiClient, assertOk } from "~/lib/api-client";
import type { AccountBidsResponse } from "./types";

type MyBidsOptions = {
  all?: boolean;
  limit?: number;
  page?: number;
};

export const accountApi = {
  async myBids({
    all = false,
    limit = 20,
    page = 1,
  }: MyBidsOptions = {}): Promise<AccountBidsResponse> {
    const params = new URLSearchParams({
      all: String(all),
      limit: String(limit),
      page: String(page),
    });
    const response = await apiClient.fetch(`/me/bids?${params.toString()}`);
    await assertOk(response);
    return (await response.json()) as AccountBidsResponse;
  },
};
