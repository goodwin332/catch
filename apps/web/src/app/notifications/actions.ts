"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { sessionHeaders } from "@/lib/session";
import { apiBaseURL } from "@/lib/auth";

export async function markNotificationRead(formData: FormData) {
  const notificationID = String(formData.get("notification_id") ?? "");
  if (!notificationID) {
    return;
  }
  const headers = await sessionHeaders();
  if (!headers) {
    return;
  }

  await fetch(`${apiBaseURL()}/notifications/${notificationID}/read`, {
    method: "POST",
    headers,
    cache: "no-store",
  });

  revalidatePath("/notifications");
  revalidatePath("/");
}

export async function openNotificationTarget(targetType: string, targetID: string) {
  const headers = await sessionHeaders();
  if (headers && targetType && targetID) {
    await fetch(`${apiBaseURL()}/notifications/read-target`, {
      method: "POST",
      headers,
      body: JSON.stringify({ target_type: targetType, target_id: targetID }),
      cache: "no-store",
    });
  }
  redirect(targetHref(targetType, targetID));
}

function targetHref(targetType: string, targetID: string) {
  if (targetType === "article") {
    return `/articles/${targetID}`;
  }
  if (targetType === "conversation" || targetType === "chat") {
    return `/chat/${targetID}`;
  }
  if (targetType === "report" || targetType === "moderation_submission") {
    return "/moderation";
  }
  return "/notifications";
}
