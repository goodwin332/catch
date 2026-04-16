"use client";

import { useEffect, useState } from "react";

type Props = {
  initialCount: number;
};

export function NotificationBadge({ initialCount }: Props) {
  const [count, setCount] = useState(initialCount);

  useEffect(() => {
    const source = new EventSource("/api/notifications/stream");
    let fallbackTimer: ReturnType<typeof setInterval> | null = null;

    const pollUnreadCount = async () => {
      try {
        const response = await fetch("/api/notifications/unread-count", { cache: "no-store" });
        if (!response.ok) {
          return;
        }
        const payload = (await response.json()) as { unread_total?: number };
        if (typeof payload.unread_total === "number") {
          setCount(payload.unread_total);
        }
      } catch {
        setCount((current) => current);
      }
    };

    source.addEventListener("unread-count", (event) => {
      try {
        const payload = JSON.parse(event.data) as { unread_total?: number };
        if (typeof payload.unread_total === "number") {
          setCount(payload.unread_total);
        }
      } catch {
        setCount((current) => current);
      }
    });

    source.onerror = () => {
      source.close();
      void pollUnreadCount();
      fallbackTimer = setInterval(pollUnreadCount, 15000);
    };

    return () => {
      source.close();
      if (fallbackTimer) {
        clearInterval(fallbackTimer);
      }
    };
  }, []);

  if (count <= 0) {
    return null;
  }

  return <span>{count > 99 ? "99+" : count}</span>;
}
