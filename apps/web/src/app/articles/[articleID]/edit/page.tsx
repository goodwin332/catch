import { PageShell } from "@/components/page-shell";
import { ArticleEditorForm } from "@/components/article-editor-form";
import { apiPublicBaseURL, authFetch } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { archiveDraft, saveDraft, submitDraft } from "./actions";

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
  const archive = archiveDraft.bind(null, articleID);

  return (
    <PageShell>
      <section className="editor-shell">
        <h1>Редактор статьи</h1>
        <p>Версия {draft.version}. Статус: {draft.status}.</p>
        {state?.saved ? <p className="auth-hint">Черновик сохранён.</p> : null}
        {state?.submitted ? <p className="auth-hint">Статья отправлена дальше по workflow.</p> : null}
        {state?.error ? <p className="auth-error">Не удалось выполнить действие.</p> : null}
        <ArticleEditorForm apiPublicBaseURL={apiPublicBaseURL()} archiveAction={archive} draft={draft} saveAction={save} submitAction={submit} />
      </section>
    </PageShell>
  );
}
