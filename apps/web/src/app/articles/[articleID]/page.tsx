import { notFound } from "next/navigation";
import { PageShell } from "@/components/page-shell";
import { getArticleComments, getPublicArticle } from "@/lib/api";
import { getCurrentUser } from "@/lib/auth";
import { addBookmark, createComment, createReport, editComment } from "./actions";

type ArticlePageProps = {
  params: Promise<{ articleID: string }>;
  searchParams?: Promise<{ saved?: string; reported?: string; commented?: string; comment_edit?: string }>;
};

export default async function ArticlePage({ params, searchParams }: ArticlePageProps) {
  const { articleID } = await params;
  const state = await searchParams;
  const article = await getPublicArticle(articleID);
  const comments = await getArticleComments(articleID);
  const currentUser = await getCurrentUser();

  if (!article) {
    notFound();
  }
  const save = addBookmark.bind(null, articleID);
  const report = createReport.bind(null, articleID);

  return (
    <PageShell>
      <article className="article-page">
        <div className="tags">
          {article.tags.map((tag) => (
            <span className="tag" key={tag}>
              #{tag}
            </span>
          ))}
        </div>
        <h1>{article.title}</h1>
        <p className="article-lead">{article.excerpt || "Материал Catch"}</p>
        <div className="reaction-summary" aria-label="Реакции">
          <span>За: {article.reactions_up}</span>
          <span>Против: {article.reactions_down}</span>
          <strong>Итог: {article.reaction_score}</strong>
        </div>
        {state?.saved ? <p className="auth-hint">Статья добавлена в закладки.</p> : null}
        {state?.reported === "1" ? <p className="auth-hint">Жалоба отправлена.</p> : null}
        {state?.reported === "0" ? <p className="auth-error">Не удалось отправить жалобу.</p> : null}
        {state?.commented === "1" ? <p className="auth-hint">Комментарий опубликован.</p> : null}
        {state?.commented === "0" ? <p className="auth-error">Не удалось добавить комментарий.</p> : null}
        {state?.comment_edit === "1" ? <p className="auth-hint">Комментарий обновлён.</p> : null}
        {state?.comment_edit === "0" ? <p className="auth-error">Комментарий уже нельзя редактировать.</p> : null}
        <div className="editor-actions">
          <form action={save}>
            <button className="primary-button" type="submit">
              В закладки
            </button>
          </form>
          <form action={report} className="inline-form">
            <select name="reason" aria-label="Причина жалобы">
              <option value="advertising">Реклама</option>
              <option value="profanity">Нецензурная лексика</option>
              <option value="insult">Оскорбление</option>
              <option value="fraud">Мошенничество</option>
              <option value="other">Другое</option>
            </select>
            <input name="details" placeholder="Комментарий" />
            <button className="secondary-button" type="submit">
              Пожаловаться
            </button>
          </form>
        </div>
        <div className="article-document">
          <pre>{JSON.stringify(article.content, null, 2)}</pre>
        </div>
      </article>

      <section className="comments-section" id="comments">
        <div className="page-heading">
          <div>
            <span className="eyebrow">Обсуждение</span>
            <h2>Комментарии</h2>
          </div>
          <span className="counter-pill">{comments.items.length}</span>
        </div>

        {currentUser ? (
          <form action={createComment.bind(null, articleID)} className="comment-form">
            <textarea name="body" rows={4} placeholder="Добавьте опыт, вопрос или уточнение" required />
            <button className="primary-button" type="submit">
              Отправить
            </button>
          </form>
        ) : (
          <p className="auth-hint">Войдите, чтобы участвовать в обсуждении.</p>
        )}

        <div className="comment-list">
          {comments.items.map((comment) => (
            <article className={comment.status === "deleted" ? "comment-item muted-item" : "comment-item"} key={comment.id}>
              <div>
                <strong>{comment.status === "deleted" ? "Комментарий удалён" : "Участник Catch"}</strong>
                <small>Рейтинг {comment.reaction_score}</small>
                <p>{comment.status === "deleted" ? "Текст скрыт после модерации." : comment.body}</p>
              </div>
              {currentUser?.user.id === comment.author_id && comment.status === "active" ? (
                <form action={editComment.bind(null, articleID, comment.id)} className="comment-edit-form">
                  <textarea name="body" rows={3} defaultValue={comment.body} required />
                  <button className="secondary-button" type="submit">
                    Сохранить
                  </button>
                </form>
              ) : null}
            </article>
          ))}
        </div>
      </section>
    </PageShell>
  );
}
