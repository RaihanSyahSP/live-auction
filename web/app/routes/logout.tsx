import { redirect } from "react-router";
import { logoutAction } from "~/features/auth/actions";

export { logoutAction as clientAction };

export function loader() {
  return redirect("/");
}

export default function Logout() {
  return null;
}
