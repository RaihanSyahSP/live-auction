export type PublicFooterAction = {
  label: string;
  to: string;
};

export type PublicChrome = {
  note: string;
  lotLabel?: string;
  demoTag?: string;
  footerNote?: string;
  footerActions?: PublicFooterAction[];
};
