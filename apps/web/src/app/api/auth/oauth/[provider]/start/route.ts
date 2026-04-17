import { NextRequest, NextResponse } from "next/server";
import { apiBaseURL } from "@/lib/auth";
import { parseSetCookie, splitSetCookie } from "@/lib/cookies";

type OAuthStartRouteProps = {
  params: Promise<{ provider: string }>;
};

export async function GET(request: NextRequest, { params }: OAuthStartRouteProps) {
  const { provider } = await params;
  const returnTo = request.nextUrl.searchParams.get("return_to") ?? "/";
  const response = await fetch(`${apiBaseURL()}/auth/oauth/${provider}/start?${new URLSearchParams({ return_to: returnTo }).toString()}`, {
    cache: "no-store",
    redirect: "manual",
  });

  const location = response.headers.get("location");
  if (!response.ok && response.status !== 302) {
    return NextResponse.redirect(new URL("/login?error=oauth-start-failed", request.url));
  }
  if (!location) {
    return NextResponse.redirect(new URL("/login?error=oauth-start-failed", request.url));
  }

  const redirectResponse = NextResponse.redirect(location);
  for (const value of splitSetCookie(response.headers.get("set-cookie"))) {
    const cookie = parseSetCookie(value);
    redirectResponse.cookies.set(cookie.name, cookie.value, { ...cookie.options, path: "/" });
  }
  return redirectResponse;
}
