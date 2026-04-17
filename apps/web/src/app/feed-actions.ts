"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function addArticleToDefaultBookmarks(articleID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/bookmarks/items`, {
    method: "POST",
    headers,
    body: JSON.stringify({ article_id: articleID }),
    cache: "no-store",
  });
  redirect("/bookmarks");
}
