import LoginForm from "~/features/auth/components/login-form";
import type { Route } from "./+types/login";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Login — Live Auction" },
    { name: "description", content: "Log in to your account" },
  ];
}

export default function Login() {

  return (
    <LoginForm />
  );
}
