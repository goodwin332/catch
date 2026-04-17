"use client";

import { useEffect, useMemo, useState } from "react";
import type { components } from "@/lib/api-types";

type ChatMessage = components["schemas"]["ChatMessage"];

type ChatThreadProps = {
  conversationID: string;
  initialMessages: ChatMessage[];
  currentUserID: string;
};

export function ChatThread({ conversationID, currentUserID, initialMessages }: ChatThreadProps) {
  const [messages, setMessages] = useState(initialMessages);
  const lastMessageID = useMemo(() => messages.at(-1)?.id ?? "", [messages]);

  useEffect(() => {
    const params = new URLSearchParams();
    if (lastMessageID) {
      params.set("after_id", lastMessageID);
    }
    const source = new EventSource(`/api/chat/conversations/${conversationID}/messages/stream?${params.toString()}`);
    source.addEventListener("message", (event) => {
      const message = JSON.parse(event.data) as ChatMessage;
      setMessages((current) => {
        if (current.some((item) => item.id === message.id)) {
          return current;
        }
        return [...current, message];
      });
    });
    return () => source.close();
  }, [conversationID, lastMessageID]);

  return (
    <div className="chat-messages" aria-live="polite">
      {messages.length === 0 ? <p className="chat-empty">Сообщений пока нет.</p> : null}
      {messages.map((message) => (
        <p className={message.sender_id === currentUserID ? "chat-message-own" : ""} key={message.id}>
          <strong>{message.sender_id}</strong>
          <span>{message.body}</span>
          <small>{message.read_at ? "Прочитано" : message.status}</small>
        </p>
      ))}
    </div>
  );
}
