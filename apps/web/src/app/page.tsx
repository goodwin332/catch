import { AppHeader } from "@/components/app-header";
import { ArticleCard } from "@/components/article-card";
import { getPopularFeed, getPublicFeed } from "@/lib/api";
import { fallbackArticles } from "@/lib/sample-data";

type HomePageProps = {
  searchParams?: Promise<{ cursor?: string; feed?: string }>;
};

export default async function HomePage({ searchParams }: HomePageProps) {
  const params = await searchParams;
  const isPopular = params?.feed === "popular";
  const feed = isPopular ? await getPopularFeed() : await getPublicFeed(params?.cursor);
  const articles = feed.items.length > 0 ? feed.items : fallbackArticles;

  return (
    <div className="shell">
      <AppHeader />
      <main className="page">
        <section aria-labelledby="feed-title">
          <div className="tabs">
            <a className={`tab ${isPopular ? "" : "tab-active"}`} href="/">
              Свежее
            </a>
            <a className={`tab ${isPopular ? "tab-active" : ""}`} href="/?feed=popular">
              Популярное
            </a>
            <a className="tab" href="/login">
              Подписки
            </a>
          </div>
          <h1 id="feed-title" style={{ position: "absolute", left: "-10000px" }}>
            Лента статей
          </h1>
          <div className="feed">
            {articles.map((article, index) => (
              <ArticleCard article={article} index={index} key={article.id} />
            ))}
          </div>
          {!isPopular && feed.next_cursor ? (
            <a className="pagination-link" href={`/?cursor=${encodeURIComponent(feed.next_cursor)}`}>
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
