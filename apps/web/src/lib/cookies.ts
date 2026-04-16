export function splitSetCookie(value: string | null): string[] {
  if (!value) {
    return [];
  }
  return value.split(/,(?=\s*[^;,=\s]+=[^;,]+)/g).map((cookie) => cookie.trim());
}

export function parseSetCookie(value: string) {
  const [pair, ...attributes] = value.split(";").map((part) => part.trim());
  const separator = pair.indexOf("=");
  const name = pair.slice(0, separator);
  const cookieValue = pair.slice(separator + 1);
  const options: {
    path?: string;
    httpOnly?: boolean;
    sameSite?: "lax" | "strict" | "none";
    secure?: boolean;
    maxAge?: number;
  } = {};

  for (const attribute of attributes) {
    const [rawKey, rawValue] = attribute.split("=");
    const key = rawKey.toLowerCase();
    if (key === "path") {
      options.path = rawValue;
    }
    if (key === "httponly") {
      options.httpOnly = true;
    }
    if (key === "secure") {
      options.secure = true;
    }
    if (key === "samesite") {
      const sameSite = rawValue?.toLowerCase();
      if (sameSite === "lax" || sameSite === "strict" || sameSite === "none") {
        options.sameSite = sameSite;
      }
    }
    if (key === "max-age") {
      const maxAge = Number.parseInt(rawValue, 10);
      if (Number.isFinite(maxAge)) {
        options.maxAge = maxAge;
      }
    }
  }

  return { name, value: cookieValue, options };
}
