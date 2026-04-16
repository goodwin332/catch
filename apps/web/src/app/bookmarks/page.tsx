import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";

type BookmarkedArticles = components["schemas"]["BookmarkedArticles"];

export default async function BookmarksPage() {
  const response = await authFetch("/bookmarks/items?limit=30");
  const data = response?.ok ? ((await response.json()) as BookmarkedArticles) : { items: [] };

  return (
    <PageShell>
      {data.items.length === 0 ? (
        <EmptyState title="Закладок пока нет" text="Сохраняйте статьи, чтобы быстро вернуться к ним перед поездкой." />
      ) : (
        <section className="list-page">
          <h1>Закладки</h1>
          <div className="compact-list">
            {data.items.map((item) => (
              <a href={`/articles/${item.article_id}`} key={`${item.list_id}-${item.article_id}`}>
                <strong>{item.title}</strong>
                <span>{item.excerpt}</span>
              </a>
            ))}
          </div>
        </section>
      )}
    </PageShell>
  );
}
