import { index, layout, type RouteConfig, route } from "@react-router/dev/routes";

export default [
  index("routes/home.tsx"),
  layout(
    "features/auth/layout.tsx", [
      route("login", "routes/login.tsx")
    ]
  )
] satisfies RouteConfig;
