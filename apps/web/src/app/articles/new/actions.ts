"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import { sessionHeaders } from "@/lib/session";

export async function createDraft(formData: FormData) {
  const title = String(formData.get("title") || "").trim();
  const body = String(formData.get("body") || "").trim();
  const tags = String(formData.get("tags") || "")
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);

  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }

  const response = await fetch(`${apiBaseURL()}/articles/drafts`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      title,
      tags,
      content: {
        type: "catch.article",
        version: 1,
        blocks: [
          {
            id: "intro",
            type: "paragraph",
            text: body || "Текст статьи",
          },
        ],
      },
    }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect("/articles/new?error=create-failed");
  }

  const draft = (await response.json()) as { id: string };
  redirect(`/articles/${draft.id}/edit`);
}
