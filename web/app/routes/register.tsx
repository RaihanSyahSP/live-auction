import { useActionData, useNavigation } from "react-router";
import { registerAction } from "~/features/auth/actions";
import { authChrome } from "~/features/auth/chrome";
import RegisterForm from "~/features/auth/components/register-form";

export { registerAction as clientAction };

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
  const actionData = useActionData<typeof registerAction>();
  const navigation = useNavigation();

  return (
    <RegisterForm
      error={actionData?.error}
      errors={actionData?.errors}
      fields={actionData?.fields}
      isSubmitting={navigation.state === "submitting"}
    />
  );
}
