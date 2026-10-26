export function parseCookies(request: Request): Record<string, string> {
  const header = request.headers.get("Cookie");
  if (!header) {
    return {};
  }

  return Object.fromEntries(
    header.split(";").map((cookie) => {
      const [key, ...rest] = cookie.trim().split("=");
      return [key, decodeURIComponent(rest.join("="))];
    }),
  );
}

export function getCookie(request: Request, name: string): string | null {
  return parseCookies(request)[name] ?? null;
}
