import { authChrome } from "~/features/auth/chrome";
import LoginForm from "~/features/auth/components/login-form";

export const handle = {
  authChrome: authChrome.login,
};

export function meta() {
  return [
    { title: "Masuk: LELANG" },
    { name: "description", content: "Masuk ke ruang lelang elektronik bekas." },
  ];
}

export default function Login() {
  return <LoginForm />;
}
