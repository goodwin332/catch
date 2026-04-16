import { PageShell } from "@/components/page-shell";
import { createDraft } from "./actions";

type NewArticlePageProps = {
  searchParams?: Promise<{ error?: string }>;
};

export default async function NewArticlePage({ searchParams }: NewArticlePageProps) {
  const params = await searchParams;

  return (
    <PageShell>
      <section className="editor-shell">
        <h1>Новая статья</h1>
        <p>Черновик сохранится в Catch и будет готов к дальнейшему редактированию блоками.</p>
        {params?.error ? <p className="auth-error">Не удалось создать черновик. Проверьте заголовок и сессию.</p> : null}
        <form action={createDraft} className="editor-form">
          <label>
            Заголовок
            <input name="title" minLength={3} maxLength={160} required placeholder="Например: Весенняя разведка на малой реке" />
          </label>
          <label>
            Текст
            <textarea name="body" rows={10} placeholder="Набросайте первый блок статьи..." />
          </label>
          <label>
            Теги
            <input name="tags" placeholder="рыбалка, маршрут, снасти" />
          </label>
          <button className="primary-button" type="submit">
            Создать черновик
          </button>
        </form>
      </section>
    </PageShell>
  );
}
