import { notFound } from "next/navigation";
import { PageShell } from "@/components/page-shell";
import { getPublicProfile } from "@/lib/api";

type ProfilePageProps = {
  params: Promise<{ username: string }>;
};

export default async function ProfilePage({ params }: ProfilePageProps) {
  const { username } = await params;
  const profile = await getPublicProfile(username);

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
      </section>
    </PageShell>
  );
}
