import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";
import { apiBaseURL } from "@/lib/auth";
import { parseSetCookie, splitSetCookie } from "@/lib/cookies";

type OAuthCallbackRouteProps = {
  params: Promise<{ provider: string }>;
};

export async function GET(request: NextRequest, { params }: OAuthCallbackRouteProps) {
  const { provider } = await params;
  const cookieStore = await cookies();
  const cookieHeader = cookieStore
    .getAll()
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");
  const query = request.nextUrl.searchParams.toString();
  const response = await fetch(`${apiBaseURL()}/auth/oauth/${provider}/callback?${query}`, {
    cache: "no-store",
    headers: cookieHeader ? { Cookie: cookieHeader } : {},
    redirect: "manual",
  });

  if (!response.ok && response.status !== 302) {
    return NextResponse.redirect(new URL("/login?error=oauth-callback-failed", request.url));
  }

  const location = safeReturnPath(response.headers.get("location"));
  const redirectResponse = NextResponse.redirect(new URL(location, request.url));
  for (const value of splitSetCookie(response.headers.get("set-cookie"))) {
    const cookie = parseSetCookie(value);
    redirectResponse.cookies.set(cookie.name, cookie.value, { ...cookie.options, path: "/" });
  }
  return redirectResponse;
}

function safeReturnPath(value: string | null) {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return "/";
  }
  return value;
}
