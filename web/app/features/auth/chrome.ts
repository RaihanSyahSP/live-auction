export type AuthChrome = {
  headerNote: string;
  footerNote: string;
  footerAction: {
    label: string;
    to: string;
  };
};

export const authChrome = {
  login: {
    headerNote: "Masuk ke ruang lelang",
    footerNote: "Masuk untuk memasang bid dan mengikuti lelang.",
    footerAction: {
      label: "Buat akun baru",
      to: "/register",
    },
  },
  register: {
    headerNote: "Buat akun pembeli",
    footerNote: "Daftar untuk mulai mengikuti lelang.",
    footerAction: {
      label: "Sudah punya akun? Masuk",
      to: "/login",
    },
  },
} satisfies Record<"login" | "register", AuthChrome>;
