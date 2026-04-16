import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { approveSubmission, rejectSubmission } from "./actions";

type ModerationSubmissionList = components["schemas"]["ModerationSubmissionList"];

export default async function ModerationPage() {
  const response = await authFetch("/moderation/submissions?limit=30");
  const data = response?.ok ? ((await response.json()) as ModerationSubmissionList) : { items: [] };

  return (
    <PageShell>
      {data.items.length === 0 ? (
        <EmptyState title="Очередь модерации пуста" text="Новые материалы появятся здесь после отправки авторами." />
      ) : (
        <section className="list-page">
          <h1>Модерация</h1>
          <div className="compact-list">
            {data.items.map((item) => (
              <div className="compact-list-item" key={item.id}>
                <strong>Статья {item.article_id}</strong>
                <span>Статус: {item.status}</span>
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
              </div>
            ))}
          </div>
        </section>
      )}
    </PageShell>
  );
}
