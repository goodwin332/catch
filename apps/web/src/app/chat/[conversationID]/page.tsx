import { PageShell } from "@/components/page-shell";
import { ChatThread } from "@/components/chat-thread";
import { authFetch, getCurrentUser } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { markConversationRead, sendMessage } from "../actions";

type MessageList = components["schemas"]["ChatMessageList"];
type ChatConversationList = components["schemas"]["ChatConversationList"];

type ChatConversationPageProps = {
  params: Promise<{ conversationID: string }>;
};

export default async function ChatConversationPage({ params }: ChatConversationPageProps) {
  const { conversationID } = await params;
  const [currentUser, conversationsResponse, response] = await Promise.all([
    getCurrentUser(),
    authFetch("/chat/conversations?limit=30"),
    authFetch(`/chat/conversations/${conversationID}/messages?limit=50`),
  ]);
  const conversations = conversationsResponse?.ok ? ((await conversationsResponse.json()) as ChatConversationList).items : [];
  const data = response?.ok ? ((await response.json()) as MessageList) : { items: [] };

  return (
    <PageShell>
      <section className="chat-layout">
        <aside className="chat-sidebar" aria-label="Диалоги">
          <h2>Диалоги</h2>
          {conversations.map((item) => (
            <a className={item.id === conversationID ? "chat-dialog-active" : ""} href={`/chat/${item.id}`} key={item.id}>
              <strong>{item.id.slice(0, 8)}</strong>
              <span>Непрочитано: {item.unread_count}</span>
            </a>
          ))}
        </aside>
        <div className="chat-main">
          <div className="page-heading">
            <div>
              <span className="eyebrow">Личные сообщения</span>
              <h1>Диалог</h1>
            </div>
            <form action={markConversationRead.bind(null, conversationID)}>
              <button className="secondary-button" type="submit">
                Всё прочитано
              </button>
            </form>
          </div>
          <ChatThread conversationID={conversationID} currentUserID={currentUser?.user.id ?? ""} initialMessages={data.items} />
          <form action={sendMessage.bind(null, conversationID)} className="editor-form">
            <label>
              Сообщение
              <textarea name="body" rows={4} required />
            </label>
            <button className="primary-button" type="submit">
              Отправить
            </button>
          </form>
        </div>
      </section>
    </PageShell>
  );
}
