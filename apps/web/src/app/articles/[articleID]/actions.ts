"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function addBookmark(articleID: string) {
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
  redirect(`/articles/${articleID}?saved=1`);
}

export async function createReport(articleID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const reason = String(formData.get("reason") || "other");
  const details = String(formData.get("details") || "").trim();
  const response = await fetch(`${apiBaseURL()}/reports`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      target_type: "article",
      target_id: articleID,
      reason,
      details,
    }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}?reported=${response.ok ? "1" : "0"}`);
}

export async function createComment(articleID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const body = String(formData.get("body") || "").trim();
  const response = await fetch(`${apiBaseURL()}/articles/${articleID}/comments`, {
    method: "POST",
    headers,
    body: JSON.stringify({ body }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}?commented=${response.ok ? "1" : "0"}#comments`);
}

export async function editComment(articleID: string, commentID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const body = String(formData.get("body") || "").trim();
  const response = await fetch(`${apiBaseURL()}/comments/${commentID}`, {
    method: "PATCH",
    headers,
    body: JSON.stringify({ body }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}?comment_edit=${response.ok ? "1" : "0"}#comments`);
}
