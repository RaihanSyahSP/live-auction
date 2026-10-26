import {
  index,
  layout,
  type RouteConfig,
  route,
} from "@react-router/dev/routes";

export default [
  layout("components/layout/app-shell.tsx", [
    layout("components/layout/public-layout.tsx", [
      index("routes/home.tsx"),
      route("results", "routes/results.tsx"),
    ]),
    layout("features/auth/layout.tsx", [
      route("login", "routes/login.tsx"),
      route("register", "routes/register.tsx"),
    ]),
    layout("features/account/layout.tsx", [
      route("account", "routes/account.tsx"),
    ]),
    layout("features/admin/layout.tsx", [route("admin", "routes/admin.tsx")]),
  ]),
] satisfies RouteConfig;
