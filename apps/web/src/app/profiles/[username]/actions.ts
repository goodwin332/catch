"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function followAuthor(username: string, authorID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/subscriptions/${authorID}`, {
    method: "POST",
    headers,
    body: JSON.stringify({}),
    cache: "no-store",
  });
  redirect(`/profiles/${encodeURIComponent(username)}?followed=1`);
}

export async function unfollowAuthor(username: string, authorID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/subscriptions/${authorID}`, {
    method: "DELETE",
    headers,
    body: JSON.stringify({}),
    cache: "no-store",
  });
  redirect(`/profiles/${encodeURIComponent(username)}?unfollowed=1`);
}

export async function startChatWithAuthor(username: string, authorID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const response = await fetch(`${apiBaseURL()}/chat/conversations`, {
    method: "POST",
    headers,
    body: JSON.stringify({ recipient_id: authorID }),
    cache: "no-store",
  });
  if (!response.ok) {
    redirect(`/profiles/${encodeURIComponent(username)}?chat=0`);
  }
  const conversation = (await response.json()) as { id: string };
  redirect(`/chat/${conversation.id}`);
}
