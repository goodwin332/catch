import { EmptyState } from "@/components/empty-state";
import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { startConversation } from "./actions";

type ChatConversationList = components["schemas"]["ChatConversationList"];

export default async function ChatPage() {
  const response = await authFetch("/chat/conversations?limit=30");
  const data = response?.ok ? ((await response.json()) as ChatConversationList) : { items: [] };

  return (
    <PageShell>
      <section className="list-page">
        <h1>Чат</h1>
        <form action={startConversation} className="inline-form">
          <input name="recipient_id" placeholder="ID пользователя" />
          <button className="primary-button" type="submit">
            Начать диалог
          </button>
        </form>
        {data.items.length === 0 ? (
          <EmptyState title="Диалогов пока нет" text="Личные сообщения появятся здесь после первого разговора." />
        ) : (
          <div className="compact-list">
            {data.items.map((item) => (
              <a href={`/chat/${item.id}`} key={item.id}>
                <strong>Диалог {item.id}</strong>
                <span>Непрочитано: {item.unread_count}</span>
              </a>
            ))}
          </div>
        )}
      </section>
    </PageShell>
  );
}
