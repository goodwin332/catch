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

export async function createThread(submissionID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  const response = await fetch(`${apiBaseURL()}/moderation/submissions/${submissionID}/threads`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      block_id: String(formData.get("block_id") || "").trim(),
      body: String(formData.get("body") || "").trim(),
    }),
    cache: "no-store",
  });
  redirect(`/moderation?thread=${response.ok ? "1" : "0"}`);
}

export async function resolveThread(threadID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/moderation/threads/${threadID}/resolve`, {
    method: "POST",
    headers,
    body: JSON.stringify({}),
    cache: "no-store",
  });
  redirect("/moderation");
}

export async function reopenThread(threadID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/moderation/threads/${threadID}/reopen`, {
    method: "POST",
    headers,
    body: JSON.stringify({ reason: String(formData.get("reason") || "").trim() }),
    cache: "no-store",
  });
  redirect("/moderation");
}

export async function decideReport(reportID: string, decision: "accept" | "reject") {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }
  await fetch(`${apiBaseURL()}/reports/${reportID}/decisions`, {
    method: "POST",
    headers,
    body: JSON.stringify({ decision }),
    cache: "no-store",
  });
  redirect("/moderation?tab=reports");
}
