import { apiClient, assertOk } from "~/lib/api-client";
import type { LoginSchema, RegisterSchema } from "./schema";
import type { User } from "./types";

export const authApi = {
  async login(input: LoginSchema, request?: Request): Promise<User> {
    const response = await apiClient.fetch("/auth/login", {
      method: "POST",
      body: JSON.stringify(input),
      _request: request,
    });

    await assertOk(response);
    return (await response.json()) as User;
  },

  async register(input: RegisterSchema, request?: Request): Promise<User> {
    const body = {
      email: input.email,
      password: input.password,
      display_name: input.displayName,
    };
    const response = await apiClient.fetch("/auth/register", {
      method: "POST",
      body: JSON.stringify(body),
      _request: request,
    });

    await assertOk(response);
    return (await response.json()) as User;
  },

  async me(request?: Request): Promise<User | null> {
    const response = await apiClient.fetch("/auth/me", {
      _request: request,
    });

    if (response.status === 401) {
      return null;
    }

    await assertOk(response);
    return (await response.json()) as User;
  },

  async refresh(request?: Request): Promise<User | null> {
    const response = await apiClient.fetch("/auth/refresh", {
      method: "POST",
      _request: request,
    });

    if (!response.ok) {
      return null;
    }

    return (await response.json()) as User;
  },

  async logout(request?: Request): Promise<boolean> {
    const response = await apiClient.fetch("/auth/logout", {
      method: "POST",
      _request: request,
    });

    return response.ok;
  },
};
