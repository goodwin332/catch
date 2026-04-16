import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { sendMessage } from "../actions";

type MessageList = components["schemas"]["ChatMessageList"];

type ChatConversationPageProps = {
  params: Promise<{ conversationID: string }>;
};

export default async function ChatConversationPage({ params }: ChatConversationPageProps) {
  const { conversationID } = await params;
  const response = await authFetch(`/chat/conversations/${conversationID}/messages?limit=50`);
  const data = response?.ok ? ((await response.json()) as MessageList) : { items: [] };

  return (
    <PageShell>
      <section className="list-page">
        <h1>Диалог</h1>
        <div className="chat-messages">
          {data.items.map((message) => (
            <p key={message.id}>
              <strong>{message.sender_id}</strong>
              <span>{message.body}</span>
            </p>
          ))}
        </div>
        <form action={sendMessage.bind(null, conversationID)} className="editor-form">
          <label>
            Сообщение
            <textarea name="body" rows={4} required />
          </label>
          <button className="primary-button" type="submit">
            Отправить
          </button>
        </form>
      </section>
    </PageShell>
  );
}
