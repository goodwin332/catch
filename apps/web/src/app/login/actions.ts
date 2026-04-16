"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { parseSetCookie, splitSetCookie } from "@/lib/cookies";

export async function devLogin() {
  const response = await fetch(`${apiBaseURL()}/dev/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: "dev@catch.local" }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect("/login?error=dev-login-unavailable");
  }

  const cookieStore = await cookies();
  for (const value of splitSetCookie(response.headers.get("set-cookie"))) {
    const cookie = parseSetCookie(value);
    cookieStore.set(cookie.name, cookie.value, cookie.options);
  }

  redirect("/");
}

export async function requestEmailCode(formData: FormData) {
  const email = String(formData.get("email") || "").trim();
  if (!email) {
    redirect("/login?error=email-required");
  }

  const response = await fetch(`${apiBaseURL()}/auth/email/request-code`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect("/login?error=email-request-failed");
  }

  const payload = (await response.json()) as { dev_code?: string };
  const query = new URLSearchParams({ email });
  if (payload.dev_code) {
    query.set("dev_code", payload.dev_code);
  }
  redirect(`/login?${query.toString()}`);
}

export async function verifyEmailCode(formData: FormData) {
  const email = String(formData.get("email") || "").trim();
  const code = String(formData.get("code") || "").trim();
  if (!email || !code) {
    redirect(`/login?${new URLSearchParams({ email, error: "code-required" }).toString()}`);
  }

  const response = await fetch(`${apiBaseURL()}/auth/email/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect(`/login?${new URLSearchParams({ email, error: "code-invalid" }).toString()}`);
  }

  const cookieStore = await cookies();
  for (const value of splitSetCookie(response.headers.get("set-cookie"))) {
    const cookie = parseSetCookie(value);
    cookieStore.set(cookie.name, cookie.value, cookie.options);
  }

  redirect("/");
}

export async function logout() {
  const cookieStore = await cookies();
  const session = cookieStore.get("catch_session");
  const csrf = cookieStore.get("catch_csrf");

  if (session && csrf) {
    await fetch(`${apiBaseURL()}/auth/logout`, {
      method: "POST",
      headers: {
        Cookie: `catch_session=${session.value}; catch_csrf=${csrf.value}`,
        "X-CSRF-Token": csrf.value,
      },
      cache: "no-store",
    });
  }

  cookieStore.delete("catch_session");
  cookieStore.delete("catch_csrf");
  redirect("/login");
}
