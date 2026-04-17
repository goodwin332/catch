"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function createBookmarkList(formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const name = String(formData.get("name") || "").trim();
  const response = await fetch(`${apiBaseURL()}/bookmarks/lists`, {
    method: "POST",
    headers,
    body: JSON.stringify({ name }),
    cache: "no-store",
  });
  redirect(`/bookmarks?created=${response.ok ? "1" : "0"}`);
}

export async function removeBookmark(articleID: string, listID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const response = await fetch(`${apiBaseURL()}/bookmarks/items`, {
    method: "DELETE",
    headers,
    body: JSON.stringify({ article_id: articleID, list_id: listID }),
    cache: "no-store",
  });
  redirect(`/bookmarks?removed=${response.ok ? "1" : "0"}`);
}
