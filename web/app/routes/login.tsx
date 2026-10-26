import { useActionData, useNavigation } from "react-router";
import { loginAction } from "~/features/auth/actions";
import { authChrome } from "~/features/auth/chrome";
import LoginForm from "~/features/auth/components/login-form";

export { loginAction as clientAction };

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
  const actionData = useActionData<typeof loginAction>();
  const navigation = useNavigation();

  return (
    <LoginForm
      error={actionData?.error}
      errors={actionData?.errors}
      fields={actionData?.fields}
      isSubmitting={navigation.state === "submitting"}
    />
  );
}
