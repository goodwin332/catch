"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function updateProfile(formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }

  const response = await fetch(`${apiBaseURL()}/profile/me`, {
    method: "PATCH",
    headers,
    body: JSON.stringify({
      username: value(formData, "username"),
      display_name: value(formData, "display_name"),
      avatar_url: value(formData, "avatar_url"),
      birth_date: value(formData, "birth_date"),
      bio: value(formData, "bio"),
      boat: value(formData, "boat"),
      country_code: value(formData, "country_code"),
      country_name: value(formData, "country_name"),
      city_name: value(formData, "city_name"),
    }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect("/profile/me?error=save-failed");
  }

  redirect("/profile/me?saved=1");
}

function value(formData: FormData, key: string) {
  return String(formData.get(key) || "").trim();
}
