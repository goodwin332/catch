import { AppHeader } from "@/components/app-header";
import { ArticleCard } from "@/components/article-card";
import { getPopularFeed, getPublicFeed } from "@/lib/api";
import type { ArticleListResponse } from "@/lib/api";
import { authFetch, getCurrentUser } from "@/lib/auth";
import { fallbackArticles } from "@/lib/sample-data";

type HomePageProps = {
  searchParams?: Promise<{ cursor?: string; feed?: string }>;
};

export default async function HomePage({ searchParams }: HomePageProps) {
  const params = await searchParams;
  const isPopular = params?.feed === "popular";
  const isSubscriptions = params?.feed === "subscriptions";
  const currentUser = await getCurrentUser();
  const feed = await loadFeed(isPopular, isSubscriptions, params?.cursor);
  const articles = feed.items.length > 0 ? feed.items : isSubscriptions ? [] : fallbackArticles;

  return (
    <div className="shell">
      <AppHeader />
      <main className="page">
        <section aria-labelledby="feed-title">
          <div className="tabs">
            <a className={`tab ${!isPopular && !isSubscriptions ? "tab-active" : ""}`} href="/">
              Свежее
            </a>
            <a className={`tab ${isPopular ? "tab-active" : ""}`} href="/?feed=popular">
              Популярное
            </a>
            <a className={`tab ${isSubscriptions ? "tab-active" : ""}`} href={currentUser ? "/?feed=subscriptions" : "/login"}>
              Подписки
            </a>
          </div>
          <h1 id="feed-title" style={{ position: "absolute", left: "-10000px" }}>
            Лента статей
          </h1>
          {isSubscriptions && !currentUser ? (
            <div className="empty-state feed-empty">
              <h2>Войдите, чтобы собрать свою ленту</h2>
              <p>Подписки работают от вашего профиля и показывают авторов, за которыми вы следите.</p>
              <a className="primary-button" href="/login">
                Войти
              </a>
            </div>
          ) : articles.length > 0 ? (
            <div className="feed">
              {articles.map((article, index) => (
                <ArticleCard article={article} index={index} key={article.id} />
              ))}
            </div>
          ) : (
            <div className="empty-state feed-empty">
              <h2>В подписках пока пусто</h2>
              <p>Найдите авторов через поиск и подпишитесь на тех, чей опыт вам полезен.</p>
              <a className="secondary-button" href="/search?q=%40">
                Найти авторов
              </a>
            </div>
          )}
          {!isPopular && feed.next_cursor ? (
            <a className="pagination-link" href={`/?${isSubscriptions ? "feed=subscriptions&" : ""}cursor=${encodeURIComponent(feed.next_cursor)}`}>
              Следующая страница
            </a>
          ) : null}
        </section>

        <aside className="sidebar" aria-label="Боковая колонка">
          <section className="side-card">
            <h2>Темы недели</h2>
            <div className="side-list">
              <a href="/search?q=%23%D1%8D%D1%85%D0%BE%D0%BB%D0%BE%D1%82">#эхолот</a>
              <a href="/search?q=%23%D1%81%D0%BF%D0%BB%D0%B0%D0%B2">#сплав</a>
              <a href="/search?q=%23%D0%BE%D1%85%D0%BE%D1%82%D0%B0">#охота</a>
              <a href="/search?q=%23%D0%BC%D0%B0%D1%80%D1%88%D1%80%D1%83%D1%82">#маршрут</a>
            </div>
          </section>
          <section className="side-card">
            <h2>Порог доступа</h2>
            <div className="side-list">
              <span>Комментарии доступны от -100 рейтинга.</span>
              <span>Жалобы открываются с 10 рейтинга.</span>
              <span>Публикация без модерации — с 1000 рейтинга.</span>
            </div>
          </section>
        </aside>
      </main>
    </div>
  );
}

async function loadFeed(isPopular: boolean, isSubscriptions: boolean, cursor?: string): Promise<ArticleListResponse> {
  if (isPopular) {
    return getPopularFeed();
  }
  if (!isSubscriptions) {
    return getPublicFeed(cursor);
  }
  const params = new URLSearchParams({ limit: "10" });
  if (cursor) {
    params.set("cursor", cursor);
  }
  const response = await authFetch(`/articles/feed?${params.toString()}`);
  if (!response?.ok) {
    return { items: [] };
  }
  return response.json() as Promise<ArticleListResponse>;
}
