import { PageShell } from "@/components/page-shell";
import { authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { saveDraft, submitDraft } from "./actions";

type ArticleDraft = components["schemas"]["ArticleDraft"];

type EditArticlePageProps = {
  params: Promise<{ articleID: string }>;
  searchParams?: Promise<{ error?: string; saved?: string; submitted?: string }>;
};

export default async function EditArticlePage({ params, searchParams }: EditArticlePageProps) {
  const { articleID } = await params;
  const state = await searchParams;
  const response = await authFetch(`/articles/drafts/${articleID}`);
  const draft = response?.ok ? ((await response.json()) as ArticleDraft) : null;

  if (!draft) {
    return (
      <PageShell>
        <section className="editor-shell">
          <h1>Черновик недоступен</h1>
          <p>Войдите в аккаунт автора или создайте новый материал.</p>
          <a className="primary-button" href="/articles/new">
            Новая статья
          </a>
        </section>
      </PageShell>
    );
  }

  const save = saveDraft.bind(null, articleID);
  const submit = submitDraft.bind(null, articleID);
  const body = extractBody(draft.content);
  const point = extractGeoPoint(draft.content);
  const route = extractRoute(draft.content);

  return (
    <PageShell>
      <section className="editor-shell">
        <h1>Редактор статьи</h1>
        <p>Версия {draft.version}. Статус: {draft.status}.</p>
        {state?.saved ? <p className="auth-hint">Черновик сохранён.</p> : null}
        {state?.submitted ? <p className="auth-hint">Статья отправлена дальше по workflow.</p> : null}
        {state?.error ? <p className="auth-error">Не удалось выполнить действие.</p> : null}
        <form action={save} className="editor-form">
          <label>
            Заголовок
            <input name="title" minLength={3} maxLength={160} defaultValue={draft.title} required />
          </label>
          <label>
            Текст
            <textarea name="body" rows={12} defaultValue={body} />
          </label>
          <label>
            Теги
            <input name="tags" defaultValue={draft.tags.join(", ")} />
          </label>
          <label>
            Медиафайл
            <input name="file" type="file" accept="image/jpeg,image/png,image/webp,image/gif,application/pdf" />
          </label>
          <fieldset className="editor-fieldset">
            <legend>Геоточка</legend>
            <label>
              Название точки
              <input name="point_label" defaultValue={point?.label ?? ""} placeholder="Стоянка у протоки" />
            </label>
            <div className="editor-grid">
              <label>
                Широта
                <input name="point_lat" defaultValue={point?.latitude ?? ""} inputMode="decimal" placeholder="55.751244" />
              </label>
              <label>
                Долгота
                <input name="point_lng" defaultValue={point?.longitude ?? ""} inputMode="decimal" placeholder="37.618423" />
              </label>
            </div>
          </fieldset>
          <fieldset className="editor-fieldset">
            <legend>Маршрут</legend>
            <label>
              Название маршрута
              <input name="route_title" defaultValue={route?.title ?? ""} placeholder="Утренняя петля" />
            </label>
            <label>
              Точки маршрута
              <textarea
                name="route_points"
                rows={5}
                defaultValue={route?.points ?? ""}
                placeholder={"55.751244, 37.618423, старт\n55.761244, 37.628423, финиш"}
              />
            </label>
          </fieldset>
          <div className="editor-actions">
            <button className="primary-button" type="submit">
              Сохранить
            </button>
            <button className="secondary-button" formAction={submit} type="submit">
              Отправить
            </button>
          </div>
        </form>
      </section>
    </PageShell>
  );
}

function extractBody(content: ArticleDraft["content"]) {
  const block = content.blocks.find((item) => item.type === "paragraph" && typeof item.text === "string");
  return typeof block?.text === "string" ? block.text : "";
}

function extractGeoPoint(content: ArticleDraft["content"]) {
  const block = content.blocks.find((item) => item.type === "geo_point");
  if (!block || typeof block.latitude !== "number" || typeof block.longitude !== "number") {
    return null;
  }
  return {
    label: typeof block.label === "string" ? block.label : "",
    latitude: String(block.latitude),
    longitude: String(block.longitude),
  };
}

function extractRoute(content: ArticleDraft["content"]) {
  const block = content.blocks.find((item) => item.type === "route");
  if (!block || !Array.isArray(block.points)) {
    return null;
  }
  const points = block.points
    .map((point) => {
      if (!point || typeof point !== "object") {
        return "";
      }
      const candidate = point as { latitude?: unknown; longitude?: unknown; label?: unknown };
      if (typeof candidate.latitude !== "number" || typeof candidate.longitude !== "number") {
        return "";
      }
      return [candidate.latitude, candidate.longitude, typeof candidate.label === "string" ? candidate.label : ""]
        .filter((item) => item !== "")
        .join(", ");
    })
    .filter(Boolean)
    .join("\n");
  return {
    title: typeof block.title === "string" ? block.title : "",
    points,
  };
}
