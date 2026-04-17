import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { approveSubmission, createThread, decideReport, rejectSubmission, reopenThread, resolveThread } from "./actions";

type ModerationSubmissionList = components["schemas"]["ModerationSubmissionList"];
type ReportList = components["schemas"]["ReportList"];
type ModerationThreadList = components["schemas"]["ModerationThreadList"];

type ModerationPageProps = {
  searchParams?: Promise<{ tab?: string; thread?: string }>;
};

export default async function ModerationPage({ searchParams }: ModerationPageProps) {
  const params = await searchParams;
  const tab = normalizeTab(params?.tab);
  const [submissionsResponse, reportsResponse] = await Promise.all([authFetch("/moderation/submissions?limit=30"), authFetch("/reports?limit=30")]);
  const data = submissionsResponse?.ok ? ((await submissionsResponse.json()) as ModerationSubmissionList) : { items: [] };
  const reports = reportsResponse?.ok ? ((await reportsResponse.json()) as ReportList) : { items: [] };
  const visibleReports =
    tab === "comment_reports"
      ? reports.items.filter((report) => report.target_type === "comment")
      : reports.items.filter((report) => report.target_type === "article");
  const threadEntries = await Promise.all(
    data.items.map(async (item) => {
      const response = await authFetch(`/moderation/submissions/${item.id}/threads`);
      const threads = response?.ok ? ((await response.json()) as ModerationThreadList).items : [];
      return [item.id, threads] as const;
    }),
  );
  const threadsBySubmission = new Map(threadEntries);

  return (
    <PageShell>
      <section className="list-page">
        <h1>Модерация</h1>
        <div className="tabs">
          <a className={`tab ${tab === "submissions" ? "tab-active" : ""}`} href="/moderation">
            Статьи
          </a>
          <a className={`tab ${tab === "article_reports" ? "tab-active" : ""}`} href="/moderation?tab=article_reports">
            Жалобы на статьи
          </a>
          <a className={`tab ${tab === "comment_reports" ? "tab-active" : ""}`} href="/moderation?tab=comment_reports">
            Жалобы на комментарии
          </a>
        </div>
        {params?.thread === "1" ? <p className="auth-hint">Тред модерации создан.</p> : null}
        {params?.thread === "0" ? <p className="auth-error">Не удалось создать тред.</p> : null}
        {tab === "submissions" ? (
          data.items.length === 0 ? (
            <EmptyState title="Очередь модерации пуста" text="Новые материалы появятся здесь после отправки авторами." />
          ) : (
          <div className="compact-list">
            {data.items.map((item) => (
              <div className="compact-list-item" key={item.id}>
                <strong>Статья {item.article_id}</strong>
                <span>
                  Статус: {item.status}. Автор: {item.author_id}
                </span>
                <span>
                  Одобрения: {item.approval_count}/5. Открытые треды: {item.open_thread_count}.
                </span>
                <a href={`/articles/${item.article_id}`}>Открыть статью</a>
                {(threadsBySubmission.get(item.id) ?? []).length > 0 ? (
                  <div className="moderation-thread-list">
                    {(threadsBySubmission.get(item.id) ?? []).map((thread) => (
                      <article className={thread.status === "resolved" ? "moderation-thread resolved-thread" : "moderation-thread"} key={thread.id}>
                        <strong>{thread.block_id ? `Блок ${thread.block_id}` : "Общее замечание"}</strong>
                        <p>{thread.body}</p>
                        <small>
                          {thread.status} · {thread.author_id}
                        </small>
                        {thread.status === "open" ? (
                          <form action={resolveThread.bind(null, thread.id)}>
                            <button className="secondary-button" type="submit">
                              Закрыть тред
                            </button>
                          </form>
                        ) : (
                          <form action={reopenThread.bind(null, thread.id)} className="inline-form">
                            <input name="reason" placeholder="Почему нужно вернуть" />
                            <button className="secondary-button" type="submit">
                              Открыть снова
                            </button>
                          </form>
                        )}
                      </article>
                    ))}
                  </div>
                ) : null}
                <div className="editor-actions">
                  <form action={approveSubmission.bind(null, item.id)}>
                    <button className="primary-button" type="submit">
                      Одобрить
                    </button>
                  </form>
                  <form action={rejectSubmission.bind(null, item.id)} className="inline-form">
                    <input name="reason" placeholder="Причина отклонения" />
                    <button className="secondary-button" type="submit">
                      Отклонить
                    </button>
                  </form>
                </div>
                <details className="reply-panel">
                  <summary>Создать тред по доработкам</summary>
                  <form action={createThread.bind(null, item.id)} className="comment-form">
                    <input name="block_id" placeholder="ID блока, если замечание точечное" />
                    <textarea name="body" placeholder="Что должен исправить автор" required rows={3} />
                    <button className="secondary-button" type="submit">
                      Создать тред
                    </button>
                  </form>
                </details>
              </div>
            ))}
          </div>
          )
        ) : visibleReports.length === 0 ? (
          <EmptyState title="Жалоб пока нет" text="Новые жалобы появятся здесь после отправки пользователями." />
        ) : (
          <div className="compact-list">
            {visibleReports.map((report) => (
              <div className="compact-list-item" key={report.id}>
                <strong>
                  {report.target_type === "article" ? "Статья" : "Комментарий"} {report.target_id}
                </strong>
                <span>
                  Причина: {report.reason}. Статус: {report.status}
                </span>
                <a href={report.target_type === "article" ? `/articles/${report.target_id}` : `/search?q=${encodeURIComponent(report.target_id)}`}>Открыть объект жалобы</a>
                {report.details ? <small>{report.details}</small> : null}
                <div className="editor-actions">
                  <form action={decideReport.bind(null, report.id, "accept")}>
                    <button className="primary-button" type="submit">
                      Принять
                    </button>
                  </form>
                  <form action={decideReport.bind(null, report.id, "reject")}>
                    <button className="secondary-button" type="submit">
                      Отклонить
                    </button>
                  </form>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </PageShell>
  );
}

function normalizeTab(value?: string) {
  if (value === "article_reports" || value === "comment_reports" || value === "reports") {
    return value === "reports" ? "article_reports" : value;
  }
  return "submissions";
}
