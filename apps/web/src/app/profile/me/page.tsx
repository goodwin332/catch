import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { updateProfile } from "./actions";

type PrivateProfile = components["schemas"]["PrivateProfile"];

type ProfileMePageProps = {
  searchParams?: Promise<{ error?: string; saved?: string }>;
};

export default async function ProfileMePage({ searchParams }: ProfileMePageProps) {
  const state = await searchParams;
  const response = await authFetch("/profile/me");
  const profile = response?.ok ? ((await response.json()) as PrivateProfile) : null;

  if (!profile) {
    return (
      <PageShell>
        <section className="empty-state">
          <h1>Профиль недоступен</h1>
          <p>Войдите, чтобы настроить публичную карточку автора.</p>
          <a className="primary-button" href="/login">
            Войти
          </a>
        </section>
      </PageShell>
    );
  }

  return (
    <PageShell>
      <section className="profile-page">
        <div className="profile-head">
          <img className="profile-avatar" src={profile.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${profile.user_id}`} alt="" />
          <div>
            <h1>Профиль</h1>
            <p>
              {profile.email}. Рейтинг: {profile.rating}. Роль: {profile.role}.
            </p>
          </div>
        </div>
        {state?.saved ? <p className="auth-hint">Профиль сохранён.</p> : null}
        {state?.error ? <p className="auth-error">Не удалось сохранить профиль.</p> : null}
        <form action={updateProfile} className="profile-form">
          <label>
            Никнейм
            <input defaultValue={profile.username || ""} name="username" placeholder="angler42" />
          </label>
          <label>
            Имя
            <input defaultValue={profile.display_name || ""} name="display_name" />
          </label>
          <label>
            Аватар
            <input defaultValue={profile.avatar_url || ""} name="avatar_url" placeholder="https://..." />
          </label>
          <label>
            Дата рождения
            <input defaultValue={profile.birth_date || ""} name="birth_date" type="date" />
          </label>
          <label>
            О себе
            <textarea defaultValue={profile.bio || ""} name="bio" rows={5} />
          </label>
          <label>
            Лодка / транспорт
            <input defaultValue={profile.boat || ""} name="boat" />
          </label>
          <div className="editor-grid">
            <label>
            Страна
              <input defaultValue={profile.country_name || ""} list="country-suggestions" name="country_name" />
            </label>
            <label>
              Код страны
              <input defaultValue={profile.country_code || ""} name="country_code" />
            </label>
          </div>
          <label>
            Город
            <input defaultValue={profile.city_name || ""} list="city-suggestions" name="city_name" />
          </label>
          <datalist id="country-suggestions">
            <option value="Россия" />
            <option value="Беларусь" />
            <option value="Казахстан" />
            <option value="Армения" />
            <option value="Грузия" />
          </datalist>
          <datalist id="city-suggestions">
            <option value="Москва" />
            <option value="Санкт-Петербург" />
            <option value="Казань" />
            <option value="Екатеринбург" />
            <option value="Новосибирск" />
            <option value="Нижний Новгород" />
          </datalist>
          <button className="primary-button" type="submit">
            Сохранить
          </button>
        </form>
      </section>
    </PageShell>
  );
}
