"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function addBookmark(articleID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const listID = String(formData.get("list_id") || "").trim();
  await fetch(`${apiBaseURL()}/bookmarks/items`, {
    method: "POST",
    headers,
    body: JSON.stringify({ article_id: articleID, ...(listID ? { list_id: listID } : {}) }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}?saved=1`);
}

export async function createReport(articleID: string, formData: FormData) {
  return createTargetReport(articleID, "article", articleID, formData);
}

export async function createTargetReport(articleID: string, targetType: "article" | "comment", targetID: string, formData: FormData) {
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
      target_type: targetType,
      target_id: targetID,
      reason,
      details,
    }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}?reported=${response.ok ? "1" : "0"}`);
}

export async function createComment(articleID: string, parentID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const body = String(formData.get("body") || "").trim();
  const payload = parentID ? { body, parent_id: parentID } : { body };
  const response = await fetch(`${apiBaseURL()}/articles/${articleID}/comments`, {
    method: "POST",
    headers,
    body: JSON.stringify(payload),
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

export async function setReaction(articleID: string, targetType: "article" | "comment", targetID: string, value: number) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/reactions`, {
    method: "POST",
    headers,
    body: JSON.stringify({ target_type: targetType, target_id: targetID, value }),
    cache: "no-store",
  });
  redirect(`/articles/${articleID}#${targetType === "comment" ? `comment-${targetID}` : "article-actions"}`);
}
