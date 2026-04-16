import { ArticleCard } from "@/components/article-card";
import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { searchArticles, searchPeople } from "@/lib/api";

type SearchPageProps = {
  searchParams: Promise<{ q?: string; cursor?: string }>;
};

export default async function SearchPage({ searchParams }: SearchPageProps) {
  const params = await searchParams;
  const query = (params.q ?? "").trim();
  const isPeopleSearch = query.startsWith("@");
  const results = isPeopleSearch ? { items: [] } : await searchArticles(query, params.cursor);
  const peopleResults = isPeopleSearch ? await searchPeople(query) : { items: [] };

  return (
    <PageShell>
      <section className="list-page">
        <div className="page-heading">
          <div>
            <span className="eyebrow">Поиск</span>
            <h1>{query ? `Результаты по запросу «${query}»` : "Найдите нужный опыт"}</h1>
          </div>
        </div>

        <form className="search-page-form" action="/search">
          <input name="q" defaultValue={query} placeholder="Статья, #тег или @автор" aria-label="Поисковый запрос" />
          <button className="primary-button" type="submit">
            Найти
          </button>
        </form>

        {!query ? (
          <EmptyState title="Введите запрос" text="Поиск начинается с трёх символов: тема, снасть, место или автор." />
        ) : query.length < 3 ? (
          <EmptyState title="Слишком короткий запрос" text="Введите минимум три символа, чтобы не шуметь в выдаче." />
        ) : isPeopleSearch && peopleResults.items.length === 0 ? (
          <EmptyState title="Авторы не найдены" text="Попробуйте username или имя автора без лишних символов." />
        ) : isPeopleSearch ? (
          <div className="profile-results">
            {peopleResults.items.map((profile) => (
              <a className="profile-result" href={`/profiles/${encodeURIComponent(profile.username ?? profile.user_id)}`} key={profile.user_id}>
                {profile.avatar_url ? (
                  <img src={profile.avatar_url} alt="" />
                ) : (
                  <span className="profile-result-avatar">{(profile.display_name || profile.username || "C").slice(0, 1).toUpperCase()}</span>
                )}
                <span>
                  <strong>{profile.display_name || profile.username || "Автор Catch"}</strong>
                  <small>@{profile.username || profile.user_id} · рейтинг {profile.rating}</small>
                  {profile.city_name || profile.country_name ? <em>{[profile.city_name, profile.country_name].filter(Boolean).join(", ")}</em> : null}
                </span>
              </a>
            ))}
          </div>
        ) : results.items.length === 0 ? (
          <EmptyState title="Ничего не найдено" text="Попробуйте другой тег, название водоёма или более общий запрос." />
        ) : (
          <>
            <div className="feed">
              {results.items.map((article, index) => (
                <ArticleCard article={article} index={index} key={article.id} />
              ))}
            </div>
            {results.next_cursor ? (
              <a className="pagination-link" href={`/search?q=${encodeURIComponent(query)}&cursor=${encodeURIComponent(results.next_cursor)}`}>
                Следующая страница
              </a>
            ) : null}
          </>
        )}
      </section>
    </PageShell>
  );
}
