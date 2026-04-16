import { cookies } from "next/headers";
import type { components } from "./api-types";

export type CurrentUser = components["schemas"]["CurrentUserResponse"];

const API_BASE_URL = process.env.CATCH_API_BASE_URL ?? "http://localhost:8080/api/v1";

export async function getCurrentUser(): Promise<CurrentUser | null> {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");

  if (!cookieHeader) {
    return null;
  }

  try {
    const response = await fetch(`${API_BASE_URL}/auth/me`, {
      headers: { Cookie: cookieHeader },
      cache: "no-store",
    });
    if (!response.ok) {
      return null;
    }
    return response.json() as Promise<CurrentUser>;
  } catch {
    return null;
  }
}

export function apiBaseURL() {
  return API_BASE_URL;
}

export async function authFetch(path: string, init?: RequestInit): Promise<Response | null> {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");

  if (!cookieHeader) {
    return null;
  }

  try {
    return fetch(`${API_BASE_URL}${path}`, {
      ...init,
      headers: {
        ...init?.headers,
        Cookie: cookieHeader,
      },
      cache: "no-store",
    });
  } catch {
    return null;
  }
}
