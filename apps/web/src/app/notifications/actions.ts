"use server";

import { revalidatePath } from "next/cache";
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

