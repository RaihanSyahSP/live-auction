export type AccountBidStatus = "leading" | "outbid" | "won" | "lost";

export type AccountBid = {
  id: number;
  lot: string;
  title: string;
  amount: number;
  at: string;
  status: AccountBidStatus;
};

export const demoAccount = {
  name: "Bidder Demo",
  email: "bidder@example.com",
  role: "Pembeli",
};

export const demoAccountBids: AccountBid[] = [
  {
    id: 1,
    lot: "001",
    title: "Sony Alpha A7 II",
    amount: 6_800_000,
    at: "09:55:10",
    status: "outbid",
  },
  {
    id: 2,
    lot: "003",
    title: "Canon EOS R",
    amount: 12_100_000,
    at: "09:12:44",
    status: "won",
  },
  {
    id: 3,
    lot: "004",
    title: "Nikon Z6",
    amount: 9_300_000,
    at: "08:52:10",
    status: "lost",
  },
];

export const bidStatusLabel: Record<AccountBidStatus, string> = {
  leading: "Tertinggi",
  outbid: "Terlampaui",
  won: "Menang",
  lost: "Tidak menang",
};
