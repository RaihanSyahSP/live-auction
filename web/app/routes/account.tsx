import AccountPage from "~/features/account/account-page";

export function meta() {
  return [
    { title: "Akun: LELANG" },
    {
      name: "description",
      content: "Profil dan riwayat bid akun LELANG.",
    },
  ];
}

export default function AccountRoute() {
  return <AccountPage />;
}
