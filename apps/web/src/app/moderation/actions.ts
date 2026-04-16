"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function approveSubmission(submissionID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/moderation/submissions/${submissionID}/approve`, {
    method: "POST",
    headers,
    body: JSON.stringify({}),
    cache: "no-store",
  });
  redirect("/moderation");
}

export async function rejectSubmission(submissionID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/moderation/submissions/${submissionID}/reject`, {
    method: "POST",
    headers,
    body: JSON.stringify({ reason: String(formData.get("reason") || "Материал требует доработки") }),
    cache: "no-store",
  });
  redirect("/moderation");
}
