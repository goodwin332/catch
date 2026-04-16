import type { components } from "./api-types";

export type ArticleListItem = components["schemas"]["ArticleListItem"];
export type ArticleListResponse = components["schemas"]["ArticleList"];
export type PublicArticle = components["schemas"]["PublicArticle"];
export type PublicProfile = components["schemas"]["PublicProfile"];
export type ProfileSearchResponse = components["schemas"]["ProfileSearch"];
export type BookmarkList = components["schemas"]["BookmarkList"];
export type BookmarkedArticle = components["schemas"]["BookmarkedArticle"];
export type ModerationSubmission = components["schemas"]["ModerationSubmission"];
export type ChatConversation = components["schemas"]["ChatConversation"];
export type CommentListResponse = components["schemas"]["CommentList"];

const API_BASE_URL = process.env.CATCH_API_BASE_URL ?? "http://localhost:8080/api/v1";

export async function getPublicFeed(cursor?: string): Promise<ArticleListResponse> {
  try {
    const params = new URLSearchParams({ limit: "10" });
    if (cursor) {
      params.set("cursor", cursor);
    }
    const response = await fetch(`${API_BASE_URL}/feed?${params.toString()}`, {
      next: { revalidate: 60 },
    });
    if (!response.ok) {
      return { items: [] };
    }
    return response.json() as Promise<ArticleListResponse>;
  } catch {
    return { items: [] };
  }
}

export async function getPopularFeed(): Promise<ArticleListResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/feed/popular?limit=10`, {
      next: { revalidate: 60 },
    });
    if (!response.ok) {
      return { items: [] };
    }
    return response.json() as Promise<ArticleListResponse>;
  } catch {
    return { items: [] };
  }
}

export async function getPublicArticle(id: string): Promise<PublicArticle | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/articles/${id}`, { next: { revalidate: 60 } });
    if (!response.ok) {
      return null;
    }
    return response.json() as Promise<PublicArticle>;
  } catch {
    return null;
  }
}

export async function getArticleComments(articleID: string): Promise<CommentListResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/articles/${articleID}/comments`, { next: { revalidate: 15 } });
    if (!response.ok) {
      return { items: [] };
    }
    return response.json() as Promise<CommentListResponse>;
  } catch {
    return { items: [] };
  }
}

export async function searchArticles(query: string, cursor?: string): Promise<ArticleListResponse> {
  const cleanQuery = query.trim();
  const articleQuery = cleanQuery.startsWith("#") ? cleanQuery.slice(1).trim() : cleanQuery;
  if (articleQuery.length < 3) {
    return { items: [] };
  }
  try {
    const params = new URLSearchParams({ q: cleanQuery, limit: "20" });
    if (cursor) {
      params.set("cursor", cursor);
    }
    const response = await fetch(`${API_BASE_URL}/search?${params.toString()}`, {
      next: { revalidate: 30 },
    });
    if (!response.ok) {
      return { items: [] };
    }
    return response.json() as Promise<ArticleListResponse>;
  } catch {
    return { items: [] };
  }
}

export async function searchPeople(query: string): Promise<ProfileSearchResponse> {
  const cleanQuery = query.trim().replace(/^@/, "").trim();
  if (cleanQuery.length < 2) {
    return { items: [] };
  }
  try {
    const params = new URLSearchParams({ q: cleanQuery, limit: "20" });
    const response = await fetch(`${API_BASE_URL}/search/people?${params.toString()}`, {
      next: { revalidate: 30 },
    });
    if (!response.ok) {
      return { items: [] };
    }
    return response.json() as Promise<ProfileSearchResponse>;
  } catch {
    return { items: [] };
  }
}

export async function getPublicProfile(username: string): Promise<PublicProfile | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/profiles/${username}`, { next: { revalidate: 60 } });
    if (!response.ok) {
      return null;
    }
    return response.json() as Promise<PublicProfile>;
  } catch {
    return null;
  }
}
