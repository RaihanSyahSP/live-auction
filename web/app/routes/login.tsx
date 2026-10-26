import LoginForm from "~/features/auth/components/login-form";

export function meta() {
  return [
    { title: "Masuk: LELANG" },
    { name: "description", content: "Masuk ke ruang lelang elektronik bekas." },
  ];
}

export default function Login() {
  return <LoginForm />;
}
