import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";

type ArticleDraft = components["schemas"]["ArticleDraft"];
type ArticleDraftList = components["schemas"]["ArticleDraftList"];

type MyArticlesPageProps = {
  searchParams?: Promise<{ archived?: string; q?: string; status?: ArticleDraft["status"] | "all" }>;
};

const statusLabels: Record<ArticleDraft["status"], string> = {
  draft: "Черновик",
  in_moderation: "На модерации",
  ready_to_publish: "Можно публиковать",
  published: "Опубликована",
  archived: "Архив",
  removed: "Удалена",
};

export default async function MyArticlesPage({ searchParams }: MyArticlesPageProps) {
  const state = await searchParams;
  const response = await authFetch("/articles/my?limit=50");
  const articles = response?.ok ? ((await response.json()) as ArticleDraftList).items : [];
  const selectedStatus = state?.status && state.status !== "all" ? state.status : "all";
  const query = (state?.q ?? "").trim().toLowerCase();
  const visibleArticles = articles.filter((article) => {
    const statusMatches = selectedStatus === "all" || article.status === selectedStatus;
    const queryMatches =
      !query ||
      article.title.toLowerCase().includes(query) ||
      article.tags.some((tag) => tag.toLowerCase().includes(query)) ||
      article.excerpt.toLowerCase().includes(query);
    return statusMatches && queryMatches;
  });
  const statusCounts = articles.reduce<Record<ArticleDraft["status"] | "all", number>>(
    (acc, article) => {
      acc.all += 1;
      acc[article.status] += 1;
      return acc;
    },
    { all: 0, draft: 0, in_moderation: 0, ready_to_publish: 0, published: 0, archived: 0, removed: 0 },
  );
  const tabs: Array<{ value: ArticleDraft["status"] | "all"; label: string }> = [
    { value: "all", label: "Все" },
    { value: "draft", label: "Черновики" },
    { value: "in_moderation", label: "Модерация" },
    { value: "ready_to_publish", label: "К публикации" },
    { value: "published", label: "Опубликовано" },
    { value: "archived", label: "Архив" },
  ];

  return (
    <PageShell>
      <section className="list-page">
        <div className="section-heading">
          <div>
            <h1>Мои статьи</h1>
            <p>Черновики, модерация, публикации и архив в одном списке.</p>
          </div>
          <a className="primary-button" href="/articles/new">
            Новая статья
          </a>
        </div>
        {state?.archived ? <p className="auth-hint">Статья перемещена в архив.</p> : null}
        <div className="tabs status-tabs">
          {tabs.map((tab) => (
            <a className={`tab ${selectedStatus === tab.value ? "tab-active" : ""}`} href={statusURL(tab.value, state?.q)} key={tab.value}>
              {tab.label} <span>{statusCounts[tab.value]}</span>
            </a>
          ))}
        </div>
        <form action="/articles/my" className="inline-form list-filter-form" method="get">
          {selectedStatus !== "all" ? <input name="status" type="hidden" value={selectedStatus} /> : null}
          <input defaultValue={state?.q || ""} name="q" placeholder="Поиск по названию, тегам и описанию" />
          <button className="secondary-button" type="submit">
            Найти
          </button>
        </form>
        {visibleArticles.length > 0 ? (
          <div className="compact-list">
            {visibleArticles.map((article) => (
              <a href={`/articles/${article.id}/edit`} key={article.id}>
                <strong>{article.title}</strong>
                <span>{statusLabels[article.status]}</span>
                <small>
                  Версия {article.version}
                  {article.published_at ? `, опубликована ${formatDate(article.published_at)}` : ""}
                  {article.scheduled_at ? `, запланирована ${formatDate(article.scheduled_at)}` : ""}
                </small>
              </a>
            ))}
          </div>
        ) : (
          <div className="empty-state inline-empty">
            <h2>{articles.length === 0 ? "Пока нет материалов" : "По фильтрам ничего не найдено"}</h2>
            <p>{articles.length === 0 ? "Начните с короткого отчёта о выезде, снастях или маршруте." : "Сбросьте фильтр или измените поисковую фразу."}</p>
          </div>
        )}
      </section>
    </PageShell>
  );
}

function statusURL(status: ArticleDraft["status"] | "all", q?: string) {
  const params = new URLSearchParams();
  if (status !== "all") {
    params.set("status", status);
  }
  if (q?.trim()) {
    params.set("q", q.trim());
  }
  const suffix = params.toString();
  return suffix ? `/articles/my?${suffix}` : "/articles/my";
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ru", { day: "numeric", month: "long", hour: "2-digit", minute: "2-digit" }).format(new Date(value));
}
