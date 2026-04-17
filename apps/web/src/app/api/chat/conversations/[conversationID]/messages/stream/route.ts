import { cookies } from "next/headers";
import { NextRequest } from "next/server";
import { apiBaseURL } from "@/lib/auth";

type ChatStreamRouteProps = {
  params: Promise<{ conversationID: string }>;
};

export async function GET(request: NextRequest, { params }: ChatStreamRouteProps) {
  const { conversationID } = await params;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");
  if (!cookieHeader) {
    return new Response("Unauthorized", { status: 401 });
  }

  const query = request.nextUrl.searchParams.toString();
  const upstream = await fetch(`${apiBaseURL()}/chat/conversations/${conversationID}/messages/stream${query ? `?${query}` : ""}`, {
    cache: "no-store",
    headers: { Cookie: cookieHeader },
  });

  if (!upstream.ok || !upstream.body) {
    return new Response("Stream unavailable", { status: upstream.status });
  }

  return new Response(upstream.body, {
    headers: {
      "Content-Type": "text/event-stream; charset=utf-8",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    },
  });
}
