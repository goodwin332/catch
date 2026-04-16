"use server";

import { redirect } from "next/navigation";
import { apiBaseURL } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { sessionHeaders } from "@/lib/session";

type ArticleDocument = components["schemas"]["ArticleDocument"];
type MediaFile = components["schemas"]["MediaFile"];
type RoutePoint = { latitude: number; longitude: number; label?: string };

export async function saveDraft(articleID: string, formData: FormData) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }

  const title = String(formData.get("title") || "").trim();
  const body = String(formData.get("body") || "").trim();
  const tags = parseTags(String(formData.get("tags") || ""));
  const media = await uploadMediaIfPresent(formData);
  const content = buildDocument(body, media, formData);

  const response = await fetch(`${apiBaseURL()}/articles/drafts/${articleID}`, {
    method: "PATCH",
    headers,
    body: JSON.stringify({ title, tags, content }),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect(`/articles/${articleID}/edit?error=save-failed`);
  }

  redirect(`/articles/${articleID}/edit?saved=1`);
}

export async function submitDraft(articleID: string) {
  const headers = await sessionHeaders();
  if (!headers) {
    redirect("/login");
  }

  const response = await fetch(`${apiBaseURL()}/articles/drafts/${articleID}/submit`, {
    method: "POST",
    headers,
    body: JSON.stringify({}),
    cache: "no-store",
  });

  if (!response.ok) {
    redirect(`/articles/${articleID}/edit?error=submit-failed`);
  }

  redirect(`/articles/${articleID}/edit?submitted=1`);
}

async function uploadMediaIfPresent(formData: FormData): Promise<MediaFile | null> {
  const file = formData.get("file");
  if (!(file instanceof File) || file.size === 0) {
    return null;
  }

  const headers = await sessionHeaders("");
  if (!headers) {
    redirect("/login");
  }

  const payload = new FormData();
  payload.set("file", file);
  const response = await fetch(`${apiBaseURL()}/media/files`, {
    method: "POST",
    headers,
    body: payload,
    cache: "no-store",
  });

  if (!response.ok) {
    return null;
  }
  return response.json() as Promise<MediaFile>;
}

function buildDocument(body: string, media: MediaFile | null, formData: FormData): ArticleDocument {
  const blocks: ArticleDocument["blocks"] = [
    {
      id: "body",
      type: "paragraph",
      text: body || "Текст статьи",
    },
  ];

  if (media) {
    blocks.push({
      id: `media-${media.id}`,
      type: media.mime_type === "application/pdf" ? "attachment" : "image",
      media_file_id: media.id,
      file_id: media.id,
      url: media.url,
      title: media.original_name,
    });
  }

  const point = parseGeoPoint(formData);
  if (point) {
    blocks.push(point);
  }

  const route = parseRoute(formData);
  if (route) {
    blocks.push(route);
  }

  return {
    type: "catch.article",
    version: 1,
    blocks,
  };
}

function parseTags(value: string) {
  return value
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function parseGeoPoint(formData: FormData) {
  const label = String(formData.get("point_label") || "").trim();
  const latitude = parseCoordinate(formData.get("point_lat"), -90, 90);
  const longitude = parseCoordinate(formData.get("point_lng"), -180, 180);
  if (latitude === null || longitude === null) {
    return null;
  }
  return {
    id: "geo-point",
    type: "geo_point",
    label: label || "Точка маршрута",
    latitude,
    longitude,
  };
}

function parseRoute(formData: FormData) {
  const title = String(formData.get("route_title") || "").trim();
  const rawPoints = String(formData.get("route_points") || "").trim();
  if (!rawPoints) {
    return null;
  }
  const points = rawPoints
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line): RoutePoint | null => {
      const [lat, lng, label] = line.split(",").map((item) => item.trim());
      const latitude = parseCoordinate(lat, -90, 90);
      const longitude = parseCoordinate(lng, -180, 180);
      if (latitude === null || longitude === null) {
        return null;
      }
      if (!label) {
        return { latitude, longitude };
      }
      return { latitude, longitude, label };
    })
    .filter((point): point is RoutePoint => point !== null);

  if (points.length < 2) {
    return null;
  }

  return {
    id: "route",
    type: "route",
    title: title || "Маршрут",
    points,
  };
}

function parseCoordinate(value: FormDataEntryValue | null, min: number, max: number) {
  const parsed = Number(String(value || "").replace(",", "."));
  if (!Number.isFinite(parsed) || parsed < min || parsed > max) {
    return null;
  }
  return parsed;
}
