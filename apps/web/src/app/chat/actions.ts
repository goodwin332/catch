"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function startConversation(formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const recipientID = String(formData.get("recipient_id") || "").trim();
  const response = await fetch(`${apiBaseURL()}/chat/conversations`, {
    method: "POST",
    headers,
    body: JSON.stringify({ recipient_id: recipientID }),
    cache: "no-store",
  });
  if (!response.ok) {
    redirect("/chat?error=start-failed");
  }
  const conversation = (await response.json()) as { id: string };
  redirect(`/chat/${conversation.id}`);
}

export async function sendMessage(conversationID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/chat/conversations/${conversationID}/messages`, {
    method: "POST",
    headers,
    body: JSON.stringify({ body: String(formData.get("body") || "").trim() }),
    cache: "no-store",
  });
  redirect(`/chat/${conversationID}`);
}
