import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { createBookmarkList, removeBookmark } from "./actions";

type BookmarkedArticles = components["schemas"]["BookmarkedArticles"];
type BookmarkLists = components["schemas"]["BookmarkLists"];

type BookmarksPageProps = {
  searchParams?: Promise<{ created?: string; list_id?: string; q?: string; removed?: string }>;
};

export default async function BookmarksPage({ searchParams }: BookmarksPageProps) {
  const params = await searchParams;
  const itemParams = new URLSearchParams({ limit: "50" });
  if (params?.list_id) {
    itemParams.set("list_id", params.list_id);
  }
  if (params?.q) {
    itemParams.set("q", params.q);
  }
  const [listsResponse, response] = await Promise.all([authFetch("/bookmarks/lists"), authFetch(`/bookmarks/items?${itemParams.toString()}`)]);
  const lists = listsResponse?.ok ? ((await listsResponse.json()) as BookmarkLists).items : [];
  const data = response?.ok ? ((await response.json()) as BookmarkedArticles) : { items: [] };
  const selectedListID = params?.list_id || "";

  return (
    <PageShell>
      <section className="list-page">
        <div className="section-heading">
          <div>
            <h1>Закладки</h1>
            <p>Списки, быстрый поиск и сохранённые материалы.</p>
          </div>
          <form action={createBookmarkList} className="inline-form">
            <input name="name" placeholder="Новый список" required />
            <button className="secondary-button" type="submit">
              Создать
            </button>
          </form>
        </div>
        {params?.created === "1" ? <p className="auth-hint">Список создан.</p> : null}
        {params?.created === "0" ? <p className="auth-error">Не удалось создать список.</p> : null}
        {params?.removed === "1" ? <p className="auth-hint">Закладка удалена.</p> : null}
        <div className="tabs status-tabs">
          <a className={`tab ${selectedListID === "" ? "tab-active" : ""}`} href={bookmarkListURL("", params?.q)}>
            Все
          </a>
          {lists.map((list) => (
            <a className={`tab ${selectedListID === list.id ? "tab-active" : ""}`} href={bookmarkListURL(list.id, params?.q)} key={list.id}>
              {list.name}
            </a>
          ))}
        </div>
        <form action="/bookmarks" className="inline-form" method="get">
          <select defaultValue={selectedListID} name="list_id" aria-label="Список">
            <option value="">Все списки</option>
            {lists.map((list) => (
              <option key={list.id} value={list.id}>
                {list.name}
              </option>
            ))}
          </select>
          <input defaultValue={params?.q || ""} name="q" placeholder="Поиск по закладкам" />
          <button className="secondary-button" type="submit">
            Найти
          </button>
        </form>
        {data.items.length === 0 ? (
          <EmptyState title="Закладок пока нет" text="Сохраняйте статьи, чтобы быстро вернуться к ним перед поездкой." />
        ) : (
          <div className="compact-list">
            {data.items.map((item) => (
              <div className="compact-list-row" key={`${item.list_id}-${item.article_id}`}>
                <a href={`/articles/${item.article_id}`}>
                  <strong>{item.title}</strong>
                  <span>{item.excerpt}</span>
                  <small>{item.tags.map((tag) => `#${tag}`).join(" ")}</small>
                  <small>Список: {item.list_name}</small>
                </a>
                <form action={removeBookmark.bind(null, item.article_id, item.list_id)}>
                  <button className="ghost-button" type="submit">
                    Удалить
                  </button>
                </form>
              </div>
            ))}
          </div>
        )}
      </section>
    </PageShell>
  );
}

function bookmarkListURL(listID: string, q?: string) {
  const params = new URLSearchParams();
  if (listID) {
    params.set("list_id", listID);
  }
  if (q?.trim()) {
    params.set("q", q.trim());
  }
  const suffix = params.toString();
  return suffix ? `/bookmarks?${suffix}` : "/bookmarks";
}
