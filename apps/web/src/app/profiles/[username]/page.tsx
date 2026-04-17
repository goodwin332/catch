import { notFound } from "next/navigation";
import { PageShell } from "@/components/page-shell";
import { getPublicProfile } from "@/lib/api";
import { getCurrentUser } from "@/lib/auth";
import { followAuthor, startChatWithAuthor, unfollowAuthor } from "./actions";

type ProfilePageProps = {
  params: Promise<{ username: string }>;
  searchParams?: Promise<{ chat?: string; followed?: string; unfollowed?: string }>;
};

export default async function ProfilePage({ params, searchParams }: ProfilePageProps) {
  const { username } = await params;
  const state = await searchParams;
  const profile = await getPublicProfile(username);
  const currentUser = await getCurrentUser();

  if (!profile) {
    notFound();
  }

  return (
    <PageShell>
      <section className="profile-page">
        <div className="profile-head">
          <img
            className="profile-avatar"
            src={profile.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${profile.user_id}`}
            alt=""
          />
          <div>
            <h1>{profile.display_name || profile.username || "Участник Catch"}</h1>
            <p>Рейтинг: {profile.rating}</p>
          </div>
        </div>
        <p className="article-lead">{profile.bio || "Автор пока не добавил рассказ о себе."}</p>
        {profile.boat ? <p className="profile-detail">Лодка: {profile.boat}</p> : null}
        {profile.city_name || profile.country_name ? <p className="profile-detail">{[profile.city_name, profile.country_name].filter(Boolean).join(", ")}</p> : null}
        {state?.followed ? <p className="auth-hint">Подписка оформлена.</p> : null}
        {state?.unfollowed ? <p className="auth-hint">Подписка отменена.</p> : null}
        {state?.chat === "0" ? <p className="auth-error">Не удалось начать чат.</p> : null}
        {currentUser && currentUser.user.id !== profile.user_id ? (
          <div className="editor-actions">
            <form action={followAuthor.bind(null, username, profile.user_id)}>
              <button className="primary-button" type="submit">
                Подписаться
              </button>
            </form>
            <form action={unfollowAuthor.bind(null, username, profile.user_id)}>
              <button className="secondary-button" type="submit">
                Отписаться
              </button>
            </form>
            <form action={startChatWithAuthor.bind(null, username, profile.user_id)}>
              <button className="secondary-button" type="submit">
                Написать
              </button>
            </form>
          </div>
        ) : null}
      </section>
    </PageShell>
  );
}
