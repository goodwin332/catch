"use client";

import { usePathname } from "next/navigation";
import { NotificationBadge } from "./notification-badge";

type MainNavProps = {
  authenticated: boolean;
  canModerate: boolean;
  unread: number;
};

export function MainNav({ authenticated, canModerate, unread }: MainNavProps) {
  const pathname = usePathname();

  return (
    <nav className="main-nav" aria-label="Основная навигация">
      <NavLink active={pathname === "/bookmarks"} href="/bookmarks">
        Закладки
      </NavLink>
      {authenticated ? (
        <NavLink active={pathname.startsWith("/articles/my")} href="/articles/my">
          Мои статьи
        </NavLink>
      ) : null}
      <NavLink active={pathname.startsWith("/chat")} href="/chat">
        Чат
      </NavLink>
      {authenticated ? (
        <NavLink active={pathname.startsWith("/profile/me")} href="/profile/me">
          Профиль
        </NavLink>
      ) : null}
      {authenticated ? (
        <NavLink active={pathname.startsWith("/notifications")} className="nav-with-badge" href="/notifications">
          Уведомления
          <NotificationBadge initialCount={unread} />
        </NavLink>
      ) : null}
      {canModerate ? (
        <NavLink active={pathname.startsWith("/moderation")} href="/moderation">
          Модерация
        </NavLink>
      ) : null}
    </nav>
  );
}

function NavLink({
  active,
  children,
  className,
  href,
}: Readonly<{ active: boolean; children: React.ReactNode; className?: string; href: string }>) {
  return (
    <a aria-current={active ? "page" : undefined} className={[className, active ? "nav-active" : ""].filter(Boolean).join(" ")} href={href}>
      {children}
    </a>
  );
}
