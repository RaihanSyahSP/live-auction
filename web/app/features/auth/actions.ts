import { type ClientActionFunctionArgs, redirect } from "react-router";
import { getErrorMessage } from "~/lib/api-client";
import { authApi } from "./api";
import { loginSchema, registerSchema } from "./schema";

export function safeNext(value: string | null) {
  return value?.startsWith("/") && !value.startsWith("//") ? value : "/";
}

export async function loginAction({ request }: ClientActionFunctionArgs) {
  const formData = await request.formData();
  const email = String(formData.get("email") ?? "");
  const parsed = loginSchema.safeParse({
    email,
    password: String(formData.get("password") ?? ""),
  });

  if (!parsed.success) {
    return {
      errors: parsed.error.flatten().fieldErrors,
      fields: { email },
    };
  }

  let user: Awaited<ReturnType<typeof authApi.login>>;
  try {
    user = await authApi.login(parsed.data);
  } catch (error) {
    return {
      error: getErrorMessage(error),
      fields: { email: parsed.data.email },
    };
  }

  const next = new URL(request.url).searchParams.get("next");
  const destination = safeNext(next);
  return redirect(
    destination === "/" && user.role === "admin" ? "/admin" : destination,
  );
}

export async function registerAction({ request }: ClientActionFunctionArgs) {
  const formData = await request.formData();
  const email = String(formData.get("email") ?? "");
  const displayName = String(formData.get("display_name") ?? "");
  const parsed = registerSchema.safeParse({
    email,
    password: String(formData.get("password") ?? ""),
    confirmPassword: String(formData.get("confirmPassword") ?? ""),
    displayName,
  });

  if (!parsed.success) {
    return {
      errors: parsed.error.flatten().fieldErrors,
      fields: { email, displayName },
    };
  }

  try {
    await authApi.register(parsed.data);
  } catch (error) {
    return {
      error: getErrorMessage(error),
      fields: {
        email: parsed.data.email,
        displayName: parsed.data.displayName,
      },
    };
  }

  const next = new URL(request.url).searchParams.get("next");
  return redirect(safeNext(next));
}

export async function logoutAction() {
  await authApi.logout();
  return redirect("/");
}
