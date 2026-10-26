import { type MiddlewareFunction, redirect } from "react-router";
import { userContext } from "~/context/auth";
import { getCookie } from "~/lib/cookies";
import { authApi } from "./api";

export const sessionMiddleware: MiddlewareFunction = async ({
  request,
  context,
}) => {
  if (!getCookie(request, "access_token")) {
    context.set(userContext, null);
    return;
  }

  const user = await authApi.me(request).catch(() => null);
  context.set(userContext, user);
};

function refreshRedirect(request: Request) {
  const url = new URL(request.url);
  const next = encodeURIComponent(`${url.pathname}${url.search}`);
  return redirect(`/refresh?next=${next}`);
}

export const authMiddleware: MiddlewareFunction = ({ request, context }) => {
  const user = context.get(userContext);

  if (!user) {
    throw refreshRedirect(request);
  }
};

export const guestMiddleware: MiddlewareFunction = ({ context }) => {
  const user = context.get(userContext);

  if (!user) {
    return;
  }

  throw redirect(user.role === "admin" ? "/admin" : "/");
};

export const adminMiddleware: MiddlewareFunction = ({ request, context }) => {
  const user = context.get(userContext);

  if (!user) {
    throw refreshRedirect(request);
  }

  if (user.role !== "admin") {
    throw redirect("/");
  }
};
