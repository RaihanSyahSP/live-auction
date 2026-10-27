export interface ApiClientOptions extends RequestInit {
  _request?: Request;
}

type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

function resolveUrl(baseURL: string, path: string, request?: Request) {
  const internalBaseURL =
    request && typeof process !== "undefined"
      ? process.env.INTERNAL_API_URL?.trim()
      : undefined;

  if (internalBaseURL) {
    return `${internalBaseURL.replace(/\/$/, "")}${path}`;
  }

  const url = `${baseURL.replace(/\/$/, "")}${path}`;

  if (/^https?:\/\//i.test(url)) {
    return url;
  }

  const origin = request
    ? new URL(request.url).origin
    : typeof window !== "undefined"
      ? window.location.origin
      : "";

  if (!origin) {
    throw new Error("Relative API URL requires a browser or request context.");
  }

  return new URL(url, origin).toString();
}

const AUTH_PATHS = new Set([
  "/auth/login",
  "/auth/register",
  "/auth/refresh",
  "/auth/logout",
]);

let refreshPromise: Promise<boolean> | null = null;

function isAuthPath(path: string) {
  return AUTH_PATHS.has(path);
}

function refreshSession(baseURL: string) {
  if (typeof window === "undefined") {
    return Promise.resolve(false);
  }

  if (!refreshPromise) {
    refreshPromise = fetch(resolveUrl(baseURL, "/auth/refresh"), {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json" },
    })
      .then((response) => response.ok)
      .catch(() => false)
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
}

export function createApiClient(baseURL: string) {
  return {
    async fetch(
      path: string,
      options: ApiClientOptions = {},
    ): Promise<Response> {
      const { _request, ...fetchOptions } = options;
      const headers = new Headers(fetchOptions.headers);

      if (!headers.has("Accept")) {
        headers.set("Accept", "application/json");
      }

      if (
        !(fetchOptions.body instanceof FormData) &&
        !headers.has("Content-Type") &&
        fetchOptions.method &&
        !["GET", "HEAD"].includes(fetchOptions.method.toUpperCase())
      ) {
        headers.set("Content-Type", "application/json");
      }

      if (_request) {
        const cookie = _request.headers.get("Cookie");
        if (cookie) {
          headers.set("Cookie", cookie);
        }
      }

      const url = resolveUrl(baseURL, path, _request);
      const response = await fetch(url, {
        ...fetchOptions,
        headers,
        credentials: "include",
      });

      if (response.status !== 401 || _request || isAuthPath(path)) {
        return response;
      }

      const refreshed = await refreshSession(baseURL);
      if (!refreshed) {
        return response;
      }

      return fetch(url, {
        ...fetchOptions,
        headers,
        credentials: "include",
      });
    },
  };
}

export async function assertOk(response: Response) {
  if (response.ok) {
    return;
  }

  const payload = (await response
    .json()
    .catch(() => null)) as ApiErrorPayload | null;
  const message =
    payload?.error?.message ?? `Request gagal (${response.status}).`;

  throw new ApiError(message, response.status, payload?.error?.code);
}

export function getErrorMessage(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }

  return "Terjadi kesalahan.";
}

export const apiClient = createApiClient(
  import.meta.env.VITE_API_URL ?? "/api",
);
