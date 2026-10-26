import { authChrome } from "~/features/auth/chrome";
import RegisterForm from "~/features/auth/components/register-form";

export const handle = {
  authChrome: authChrome.register,
};

export function meta() {
  return [
    { title: "Mendaftar: LELANG" },
    {
      name: "description",
      content: "Mendaftar ke ruang lelang elektronik bekas.",
    },
  ];
}

export default function Register() {
  return <RegisterForm />;
}
