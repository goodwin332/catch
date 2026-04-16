import { cookies } from "next/headers";
import { apiBaseURL } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET() {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");

  if (!cookieHeader) {
    return new Response("Unauthorized", { status: 401 });
  }

  const upstream = await fetch(`${apiBaseURL()}/notifications/stream`, {
    headers: { Cookie: cookieHeader },
    cache: "no-store",
  });

  if (!upstream.ok || !upstream.body) {
    return new Response(upstream.body, { status: upstream.status });
  }

  return new Response(upstream.body, {
    status: 200,
    headers: {
      "Content-Type": "text/event-stream; charset=utf-8",
      "Cache-Control": "no-cache, no-transform",
      Connection: "keep-alive",
      "X-Accel-Buffering": "no",
    },
  });
}

