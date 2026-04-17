import { logout } from "@/app/login/actions";
import { authFetch, getCurrentUser } from "@/lib/auth";
import { ThemeToggle } from "./theme-toggle";
import { MainNav } from "./main-nav";
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
        <MainNav authenticated={Boolean(currentUser)} canModerate={Boolean(currentUser?.capabilities.can_moderate)} unread={unread} />
        {currentUser?.capabilities.can_create_article ? (
          <a className="primary-button" href="/articles/new">
            Написать
          </a>
        ) : currentUser ? (
          <span className="locked-action" title="Нужен рейтинг 0 или выше">
            Написать
          </span>
        ) : null}
        {currentUser ? (
          <details className="user-menu">
            <summary>
              <img className="avatar" src={currentUser.user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${currentUser.user.id}`} alt="" />
              <span>{currentUser.user.display_name || currentUser.user.username || currentUser.user.email}</span>
            </summary>
            <div className="user-menu-panel">
              <a href={currentUser.user.username ? `/profiles/${currentUser.user.username}` : "/profile/me"}>Профиль</a>
              <ThemeToggle />
              <a href="/articles/my">Мои статьи</a>
              <a href="/bookmarks">Закладки</a>
              {currentUser.capabilities.can_moderate ? <a href="/moderation">Модерация</a> : null}
              <a href="/chat">Чат</a>
              <form action={logout}>
                <button type="submit">Выйти</button>
              </form>
            </div>
          </details>
        ) : (
          <a className="primary-button" href="/login">
            Войти
          </a>
        )}
      </div>
    </header>
  );
}
