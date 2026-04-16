import type { ArticleListItem } from "@/lib/api";
import { articleImages } from "@/lib/sample-data";

type ArticleCardProps = {
  article: ArticleListItem;
  index: number;
};

export function ArticleCard({ article, index }: ArticleCardProps) {
  const publishedAt = new Intl.DateTimeFormat("ru", {
    day: "numeric",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(article.published_at));

  return (
    <article className="article-card">
      <img className="article-cover" src={articleImages[index % articleImages.length]} alt="" />
      <div className="article-body">
        <div className="meta">
          <span className="author">
            <img
              className="author-avatar"
              src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${article.author_id}`}
              alt=""
            />
            <span>Автор Catch</span>
          </span>
          <time dateTime={article.published_at}>{publishedAt}</time>
          <span className="score-pill">Рейтинг {article.reaction_score}</span>
        </div>
        <h1 className="article-title">{article.title}</h1>
        <p className="article-excerpt">{article.excerpt}</p>
        <div className="tags" aria-label="Теги">
          {article.tags.map((tag) => (
            <a className="tag" href={`/search?q=${encodeURIComponent(`#${tag}`)}`} key={tag}>
              #{tag}
            </a>
          ))}
        </div>
      </div>
    </article>
  );
}
