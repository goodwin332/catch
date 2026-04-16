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
    return Response.json({ unread_total: 0 }, { status: 401 });
  }

  const upstream = await fetch(`${apiBaseURL()}/notifications/unread-count`, {
    headers: { Cookie: cookieHeader },
    cache: "no-store",
  });

  if (!upstream.ok) {
    return Response.json({ unread_total: 0 }, { status: upstream.status });
  }

  const payload = await upstream.json();
  return Response.json(payload);
}

