import AdminPage from "~/features/admin/admin-page";

export function meta() {
  return [
    { title: "Admin: LELANG" },
    {
      name: "description",
      content: "Halaman admin demo untuk mengelola lelang.",
    },
  ];
}

export default function AdminRoute() {
  return <AdminPage />;
}
