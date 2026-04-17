import type { ArticleListItem } from "@/lib/api";
import { articleImages } from "@/lib/sample-data";
import { addArticleToDefaultBookmarks } from "@/app/feed-actions";

type ArticleCardProps = {
  article: ArticleListItem;
  index: number;
};

export function ArticleCard({ article, index }: ArticleCardProps) {
  const publishedAt = relativeDate(article.published_at);
  const reactionTotal = article.reactions_up + article.reactions_down;
  const positivePercent = reactionTotal > 0 ? Math.round((article.reactions_up / reactionTotal) * 100) : 0;

  return (
    <article className="article-card">
      <img className="article-cover" src={article.cover_url || articleImages[index % articleImages.length]} alt="" />
      <div className="article-body">
        <div className="meta">
          <span className="author">
            <img
              className="author-avatar"
              src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${article.author_id}`}
              alt=""
            />
            <a href={`/profiles/${article.author_id}`}>Автор Catch</a>
          </span>
          <time dateTime={article.published_at}>{publishedAt}</time>
          <span className="score-pill">Рейтинг {article.reaction_score}</span>
          {reactionTotal > 0 ? (
            <span className={positivePercent >= 90 ? "ratio-pill positive-ratio" : positivePercent <= 10 ? "ratio-pill negative-ratio" : "ratio-pill"}>
              {positivePercent}%
            </span>
          ) : null}
        </div>
        <h1 className="article-title">
          <a href={`/articles/${article.id}`}>{article.title}</a>
        </h1>
        <p className="article-excerpt">{article.excerpt}</p>
        <div className="tags" aria-label="Теги">
          {article.tags.map((tag) => (
            <a className="tag" href={`/search?q=${encodeURIComponent(`#${tag}`)}`} key={tag}>
              #{tag}
            </a>
          ))}
        </div>
        <form action={addArticleToDefaultBookmarks.bind(null, article.id)} className="feed-bookmark-form">
          <button className="secondary-button" type="submit">
            В закладки
          </button>
        </form>
      </div>
    </article>
  );
}

function relativeDate(value: string) {
  const date = new Date(value);
  const diffMs = Date.now() - date.getTime();
  const minute = 60_000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (diffMs >= 0 && diffMs < hour) {
    const minutes = Math.max(1, Math.floor(diffMs / minute));
    return `${minutes} мин назад`;
  }
  if (diffMs >= 0 && diffMs < day) {
    const hours = Math.max(1, Math.floor(diffMs / hour));
    return `${hours} ч назад`;
  }
  const yesterday = new Date();
  yesterday.setDate(yesterday.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return "Вчера";
  }
  return new Intl.DateTimeFormat("ru", { day: "numeric", month: "long" }).format(date);
}
