import { logout } from "@/app/login/actions";
import { authFetch, getCurrentUser } from "@/lib/auth";
import { ThemeToggle } from "./theme-toggle";
import { NotificationBadge } from "./notification-badge";
import type { components } from "@/lib/api-types";

type UnreadCount = components["schemas"]["UnreadNotificationCount"];

export async function AppHeader() {
  const currentUser = await getCurrentUser();
  const unreadResponse = currentUser ? await authFetch("/notifications/unread-count") : null;
  const unread = unreadResponse?.ok ? ((await unreadResponse.json()) as UnreadCount).unread_total : 0;

  return (
    <header className="topbar">
      <div className="topbar-inner">
        <a className="brand" href="/">
          <span className="brand-mark">C</span>
          Catch
        </a>
        <form className="search" action="/search">
          <input name="q" placeholder="Поиск статей, #тегов или @авторов..." aria-label="Поиск" />
        </form>
        <nav className="main-nav" aria-label="Основная навигация">
          <a href="/bookmarks">Закладки</a>
          <a href="/chat">Чат</a>
          {currentUser ? (
            <a href="/notifications" className="nav-with-badge">
              Уведомления
              <NotificationBadge initialCount={unread} />
            </a>
          ) : null}
          {currentUser?.capabilities.can_moderate ? <a href="/moderation">Модерация</a> : null}
        </nav>
        <ThemeToggle />
        {currentUser?.capabilities.can_create_article ? (
          <a className="primary-button" href="/articles/new">
            Написать
          </a>
        ) : null}
        {currentUser ? (
          <form action={logout} className="user-chip">
            <img
              className="avatar"
              src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${currentUser.user.id}`}
              alt=""
            />
            <span>{currentUser.user.display_name || currentUser.user.email}</span>
            <button type="submit">Выйти</button>
          </form>
        ) : (
          <a className="primary-button" href="/login">
            Войти
          </a>
        )}
      </div>
    </header>
  );
}
