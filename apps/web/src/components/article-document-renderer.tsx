import type { components } from "@/lib/api-types";

type ArticleDocument = components["schemas"]["ArticleDocument"];
type ArticleBlock = ArticleDocument["blocks"][number] & Record<string, unknown>;

type ArticleDocumentRendererProps = {
  content: ArticleDocument;
  apiPublicBaseURL: string;
};

export function ArticleDocumentRenderer({ content, apiPublicBaseURL }: ArticleDocumentRendererProps) {
  return (
    <div className="article-rendered-document">
      {content.blocks.map((block, index) => (
        <ArticleBlockView apiPublicBaseURL={apiPublicBaseURL} block={block as ArticleBlock} key={String(block.id ?? index)} />
      ))}
    </div>
  );
}

function ArticleBlockView({ block, apiPublicBaseURL }: { block: ArticleBlock; apiPublicBaseURL: string }) {
  switch (block.type) {
    case "paragraph":
      return <p>{stringValue(block.text, "Текст статьи")}</p>;
    case "image": {
      const src = mediaSource(block, apiPublicBaseURL, true);
      return (
        <figure className="article-media">
          {src ? <img src={src} alt={stringValue(block.title, "")} /> : <div className="media-placeholder">Изображение</div>}
          {block.title ? <figcaption>{String(block.title)}</figcaption> : null}
        </figure>
      );
    }
    case "attachment": {
      const href = mediaSource(block, apiPublicBaseURL, false);
      return (
        <p className="article-attachment">
          {href ? <a href={href}>Открыть файл: {stringValue(block.title, "вложение")}</a> : stringValue(block.title, "Вложение")}
        </p>
      );
    }
    case "geo_point":
      return (
        <aside className="geo-preview">
          <strong>{stringValue(block.label, "Геоточка")}</strong>
          <span>
            {numberValue(block.latitude)}, {numberValue(block.longitude)}
          </span>
          {typeof block.radius_meters === "number" ? <small>Радиус: {Math.round(block.radius_meters)} м</small> : null}
        </aside>
      );
    case "route":
      return <RoutePreview block={block} />;
    default:
      return null;
  }
}

function RoutePreview({ block }: { block: ArticleBlock }) {
  const points = Array.isArray(block.points) ? block.points : [];
  return (
    <aside className="route-preview">
      <strong>{stringValue(block.title, "Маршрут")}</strong>
      <ol>
        {points.map((point, index) => {
          const candidate = point as Record<string, unknown>;
          return (
            <li key={`${numberValue(candidate.latitude)}-${numberValue(candidate.longitude)}-${index}`}>
              <span>{stringValue(candidate.label, `Точка ${index + 1}`)}</span>
              <small>
                {numberValue(candidate.latitude)}, {numberValue(candidate.longitude)}
              </small>
            </li>
          );
        })}
      </ol>
    </aside>
  );
}

export function mediaSource(block: ArticleBlock, apiPublicBaseURL: string, preferPreview: boolean) {
  const preferred = preferPreview ? block.preview_url : block.url;
  const fallback = preferPreview ? block.url : block.preview_url;
  const raw = typeof preferred === "string" && preferred ? preferred : typeof fallback === "string" ? fallback : "";
  if (raw) {
    return absolutize(raw, apiPublicBaseURL);
  }
  const fileID = typeof block.file_id === "string" ? block.file_id : typeof block.media_file_id === "string" ? block.media_file_id : "";
  if (!fileID) {
    return "";
  }
  return `${apiPublicBaseURL}/api/v1/media/files/${fileID}/${preferPreview ? "preview" : "content"}`;
}

function absolutize(value: string, apiPublicBaseURL: string) {
  if (value.startsWith("http://") || value.startsWith("https://")) {
    return value;
  }
  if (value.startsWith("/")) {
    return `${apiPublicBaseURL}${value}`;
  }
  return value;
}

function stringValue(value: unknown, fallback: string) {
  return typeof value === "string" && value.trim() ? value : fallback;
}

function numberValue(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value.toFixed(6) : "0.000000";
}
