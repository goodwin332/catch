import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { markNotificationRead, openNotificationTarget } from "./actions";

type NotificationList = components["schemas"]["NotificationList"];

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

export default async function NotificationsPage() {
  const response = await authFetch("/notifications?limit=50");
  const data = response?.ok ? ((await response.json()) as NotificationList) : { items: [], unread_total: 0 };

  return (
    <PageShell>
      <section className="list-page">
        <div className="page-heading">
          <div>
            <span className="eyebrow">Центр событий</span>
            <h1>Уведомления</h1>
          </div>
          <span className="counter-pill">Непрочитано: {data.unread_total}</span>
        </div>

        {data.items.length === 0 ? (
          <EmptyState title="Пока тихо" text="Здесь появятся ответы, решения модерации, жалобы и сообщения." />
        ) : (
          <div className="compact-list notification-list">
            {data.items.map((item) => (
              <article className={item.read_at ? "compact-list-item muted-item" : "compact-list-item"} key={item.id}>
                <div>
                  <strong>{item.title}</strong>
                  <span>{item.body}</span>
                  <small>
                    {formatDate(item.updated_at)} · {item.event_type}
                  </small>
                </div>
                {item.read_at ? null : (
                  <div className="editor-actions">
                    {item.target_type && item.target_id ? (
                      <form action={openNotificationTarget.bind(null, item.target_type, item.target_id)}>
                        <button className="primary-button" type="submit">
                          Открыть
                        </button>
                      </form>
                    ) : null}
                    <form action={markNotificationRead}>
                      <input type="hidden" name="notification_id" value={item.id} />
                      <button className="secondary-button" type="submit">
                        Прочитано
                      </button>
                    </form>
                  </div>
                )}
              </article>
            ))}
          </div>
        )}
      </section>
    </PageShell>
  );
}
