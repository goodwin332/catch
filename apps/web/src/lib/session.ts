import { cookies } from "next/headers";

export async function sessionHeaders(contentType = "application/json") {
  const cookieStore = await cookies();
  const session = cookieStore.get("catch_session");
  const csrf = cookieStore.get("catch_csrf");
  if (!session || !csrf) {
    return null;
  }
  const headers: Record<string, string> = {
    Cookie: `catch_session=${session.value}; catch_csrf=${csrf.value}`,
    "X-CSRF-Token": csrf.value,
  };
  if (contentType) {
    headers["Content-Type"] = contentType;
  }
  return headers;
}
