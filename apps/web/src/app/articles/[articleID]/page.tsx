import { notFound } from "next/navigation";
import { ArticleDocumentRenderer } from "@/components/article-document-renderer";
import { PageShell } from "@/components/page-shell";
import { getArticleComments, getPopularFeed, getPublicArticle } from "@/lib/api";
import type { CommentListResponse } from "@/lib/api";
import { apiPublicBaseURL, authFetch, getCurrentUser } from "@/lib/auth";
import type { CurrentUser } from "@/lib/auth";
import type { components } from "@/lib/api-types";
import { addBookmark, createComment, createReport, createTargetReport, editComment, setReaction } from "./actions";

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
  const relatedFeed = await getPopularFeed();
  const bookmarkListsResponse = currentUser ? await authFetch("/bookmarks/lists") : null;
  const bookmarkLists = bookmarkListsResponse?.ok ? ((await bookmarkListsResponse.json()) as components["schemas"]["BookmarkLists"]).items : [];

  if (!article) {
    notFound();
  }
  const save = addBookmark.bind(null, articleID);
  const report = createReport.bind(null, articleID);
  const articleReactionUp = setReaction.bind(null, articleID, "article", articleID, 1);
  const articleReactionDown = setReaction.bind(null, articleID, "article", articleID, -1);
  const commentTree = buildCommentTree(comments.items);
  const reactionTotal = article.reactions_up + article.reactions_down;
  const positivePercent = reactionTotal > 0 ? Math.round((article.reactions_up / reactionTotal) * 100) : 0;
  const relatedArticles = relatedFeed.items.filter((item) => item.id !== article.id).slice(0, 3);

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
          {reactionTotal > 0 ? <strong className={positivePercent >= 90 ? "positive-ratio" : positivePercent <= 10 ? "negative-ratio" : ""}>{positivePercent}% положительных</strong> : null}
        </div>
        {state?.saved ? <p className="auth-hint">Статья добавлена в закладки.</p> : null}
        {state?.reported === "1" ? <p className="auth-hint">Жалоба отправлена.</p> : null}
        {state?.reported === "0" ? <p className="auth-error">Не удалось отправить жалобу.</p> : null}
        {state?.commented === "1" ? <p className="auth-hint">Комментарий опубликован.</p> : null}
        {state?.commented === "0" ? <p className="auth-error">Не удалось добавить комментарий.</p> : null}
        {state?.comment_edit === "1" ? <p className="auth-hint">Комментарий обновлён.</p> : null}
        {state?.comment_edit === "0" ? <p className="auth-error">Комментарий уже нельзя редактировать.</p> : null}
        <div className="editor-actions" id="article-actions">
          <form action={articleReactionUp}>
            <button className="secondary-button" type="submit">
              Полезно
            </button>
          </form>
          <form action={articleReactionDown}>
            <button className="secondary-button" type="submit">
              Спорно
            </button>
          </form>
          <form action={save}>
            {bookmarkLists.length > 0 ? (
              <select name="list_id" aria-label="Список закладок">
                {bookmarkLists.map((list) => (
                  <option key={list.id} value={list.id}>
                    {list.name}
                  </option>
                ))}
              </select>
            ) : null}
            <button className="primary-button" type="submit">
              В закладки
            </button>
          </form>
          <ReportPanel action={report} />
        </div>
        <ArticleDocumentRenderer apiPublicBaseURL={apiPublicBaseURL()} content={article.content} />
      </article>

      {relatedArticles.length > 0 ? (
        <section className="list-page related-section">
          <span className="eyebrow">Похожие материалы</span>
          <h2>Ещё почитать</h2>
          <div className="compact-list">
            {relatedArticles.map((item) => (
              <a href={`/articles/${item.id}`} key={item.id}>
                <strong>{item.title}</strong>
                <span>{item.excerpt}</span>
              </a>
            ))}
          </div>
        </section>
      ) : null}

      <section className="comments-section" id="comments">
        <div className="page-heading">
          <div>
            <span className="eyebrow">Обсуждение</span>
            <h2>Комментарии</h2>
          </div>
          <span className="counter-pill">{comments.items.length}</span>
        </div>

        {currentUser?.capabilities.can_comment ? (
          <form action={createComment.bind(null, articleID, "")} className="comment-form">
            <textarea name="body" rows={4} placeholder="Добавьте опыт, вопрос или уточнение" required />
            <button className="primary-button" type="submit">
              Отправить
            </button>
          </form>
        ) : currentUser ? (
          <p className="auth-hint">Чтобы оставлять комментарии, нужен рейтинг не ниже -100.</p>
        ) : (
          <p className="auth-hint">Войдите, чтобы участвовать в обсуждении.</p>
        )}

        <div className="comment-list">
          {commentTree.map((comment) => (
            <CommentItem articleID={articleID} comment={comment} currentUser={currentUser} key={comment.id} />
          ))}
        </div>
      </section>
    </PageShell>
  );
}

type Comment = CommentListResponse["items"][number];
type CommentNode = Comment & { children: CommentNode[] };

function buildCommentTree(items: Comment[]) {
  const byID = new Map<string, CommentNode>();
  const roots: CommentNode[] = [];
  for (const item of items) {
    byID.set(item.id, { ...item, children: [] });
  }
  for (const node of byID.values()) {
    if (node.parent_id && byID.has(node.parent_id)) {
      byID.get(node.parent_id)?.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

function CommentItem({ articleID, comment, currentUser }: { articleID: string; comment: CommentNode; currentUser: CurrentUser | null }) {
  const canInteract = Boolean(currentUser && comment.status === "active");
  return (
    <article className={comment.status === "deleted" ? "comment-item muted-item" : "comment-item"} id={`comment-${comment.id}`}>
      <div>
        <div className="comment-meta">
          <strong>{comment.status === "deleted" ? "Комментарий удалён" : "Участник Catch"}</strong>
          <a href={`#comment-${comment.id}`}>#{comment.id.slice(0, 8)}</a>
          <small>Рейтинг {comment.reaction_score}</small>
        </div>
        <p>{comment.status === "deleted" ? "Текст скрыт после модерации." : comment.body}</p>
      </div>
      {canInteract ? (
        <div className="comment-actions">
          <form action={setReaction.bind(null, articleID, "comment", comment.id, 1)}>
            <button className="secondary-button" type="submit">
              +
            </button>
          </form>
          <form action={setReaction.bind(null, articleID, "comment", comment.id, -1)}>
            <button className="secondary-button" type="submit">
              -
            </button>
          </form>
          <ReportPanel action={createTargetReport.bind(null, articleID, "comment", comment.id)} />
        </div>
      ) : null}
      {currentUser?.user.id === comment.author_id && comment.status === "active" ? (
        <form action={editComment.bind(null, articleID, comment.id)} className="comment-edit-form">
          <textarea name="body" rows={3} defaultValue={comment.body} required />
          <button className="secondary-button" type="submit">
            Сохранить
          </button>
        </form>
      ) : null}
      {canInteract ? (
        <details className="reply-panel">
          <summary>Ответить</summary>
          <form action={createComment.bind(null, articleID, comment.id)} className="comment-form">
            <textarea name="body" rows={3} placeholder="Ответить в ветке" required />
            <button className="primary-button" type="submit">
              Отправить
            </button>
          </form>
        </details>
      ) : null}
      {comment.children.length > 0 ? (
        <div className="comment-children">
          {comment.children.map((child) => (
            <CommentItem articleID={articleID} comment={child} currentUser={currentUser} key={child.id} />
          ))}
        </div>
      ) : null}
    </article>
  );
}

function ReportPanel({ action }: { action: (formData: FormData) => void | Promise<void> }) {
  return (
    <details className="report-panel">
      <summary>Пожаловаться</summary>
      <form action={action} className="inline-form">
        <select name="reason" aria-label="Причина жалобы">
          <option value="advertising">Реклама</option>
          <option value="profanity">Нецензурная лексика</option>
          <option value="insult">Оскорбление</option>
          <option value="fraud">Мошенничество</option>
          <option value="other">Другое</option>
        </select>
        <input name="details" placeholder="Комментарий" />
        <button className="secondary-button" type="submit">
          Отправить
        </button>
      </form>
    </details>
  );
}
