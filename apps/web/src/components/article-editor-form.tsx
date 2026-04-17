"use client";

import { useMemo, useState } from "react";
import type { components } from "@/lib/api-types";
import { ArticleDocumentRenderer } from "./article-document-renderer";

type ArticleDraft = components["schemas"]["ArticleDraft"];
type ArticleDocument = components["schemas"]["ArticleDocument"];
type ArticleBlock = ArticleDocument["blocks"][number] & Record<string, unknown>;

type ArticleEditorFormProps = {
  draft: ArticleDraft;
  apiPublicBaseURL: string;
  saveAction: (formData: FormData) => void | Promise<void>;
  submitAction: (formData: FormData) => void | Promise<void>;
  archiveAction: (formData: FormData) => void | Promise<void>;
};

export function ArticleEditorForm({ draft, apiPublicBaseURL, saveAction, submitAction, archiveAction }: ArticleEditorFormProps) {
  const [title, setTitle] = useState(draft.title);
  const [body, setBody] = useState(extractBody(draft.content));
  const [tags, setTags] = useState(draft.tags.join(", "));
  const [pointLabel, setPointLabel] = useState(extractGeoPoint(draft.content)?.label ?? "");
  const [pointLat, setPointLat] = useState(extractGeoPoint(draft.content)?.latitude ?? "");
  const [pointLng, setPointLng] = useState(extractGeoPoint(draft.content)?.longitude ?? "");
  const [pointRadius, setPointRadius] = useState(extractGeoPoint(draft.content)?.radiusMeters ?? "");
  const [routeTitle, setRouteTitle] = useState(extractRoute(draft.content)?.title ?? "");
  const [routePoints, setRoutePoints] = useState(extractRoute(draft.content)?.points ?? "");
  const [publishAt, setPublishAt] = useState(formatDateTimeLocal(draft.scheduled_at ?? ""));
  const [fileName, setFileName] = useState("");
  const existingMediaBlocks = useMemo(() => extractMediaBlocks(draft.content), [draft.content]);
  const tagSuggestions = useMemo(() => ["рыбалка", "охота", "маршрут", "сплав", "эхолот", "лодка", "зимняя рыбалка", "снасти"], []);
  const previewContent = useMemo<ArticleDocument>(
    () => ({
      type: "catch.article",
      version: 1,
      blocks: [
        { id: "body-preview", type: "paragraph", text: body || "Текст статьи" },
        ...existingMediaBlocks,
        ...previewGeoBlocks(pointLabel, pointLat, pointLng, pointRadius, routeTitle, routePoints),
      ],
    }),
    [body, existingMediaBlocks, pointLabel, pointLat, pointLng, pointRadius, routeTitle, routePoints],
  );

  return (
    <form action={saveAction} className="editor-layout">
      <input name="existing_media_blocks" type="hidden" value={JSON.stringify(existingMediaBlocks)} />
      <div className="editor-form">
        <label>
          Заголовок
          <input name="title" minLength={3} maxLength={160} onChange={(event) => setTitle(event.target.value)} required value={title} />
        </label>
        <label>
          Текст
          <textarea name="body" onChange={(event) => setBody(event.target.value)} rows={14} value={body} />
        </label>
        <label>
          Теги
          <input list="catch-tag-suggestions" name="tags" onChange={(event) => setTags(event.target.value)} placeholder="рыбалка, маршрут, снасти" value={tags} />
          <datalist id="catch-tag-suggestions">
            {tagSuggestions.map((tag) => (
              <option key={tag} value={tag} />
            ))}
          </datalist>
          <span className="field-hint">До 10 тегов через запятую. Используйте устойчивые темы, чтобы поиск и рекомендации работали лучше.</span>
        </label>
        <label className="media-upload-zone">
          <span>Медиафайл</span>
          <input
            name="file"
            type="file"
            accept="image/jpeg,image/png,image/webp,image/gif,application/pdf"
            onChange={(event) => setFileName(event.target.files?.[0]?.name ?? "")}
          />
          <span className="media-upload-copy">{fileName ? `Будет добавлен файл: ${fileName}` : "Выберите файл или перетащите его в поле выбора."}</span>
          <span className="field-hint">Первое изображение в статье используется как обложка в ленте. PDF сохраняется как вложение.</span>
        </label>
        <fieldset className="editor-fieldset">
          <legend>Геоточка</legend>
          <label>
            Название точки
            <input name="point_label" onChange={(event) => setPointLabel(event.target.value)} placeholder="Стоянка у протоки" value={pointLabel} />
          </label>
          <div className="editor-grid">
            <label>
              Широта
              <input name="point_lat" inputMode="decimal" onChange={(event) => setPointLat(event.target.value)} placeholder="55.751244" value={pointLat} />
            </label>
            <label>
              Долгота
              <input name="point_lng" inputMode="decimal" onChange={(event) => setPointLng(event.target.value)} placeholder="37.618423" value={pointLng} />
            </label>
          </div>
          <label>
            Радиус, м
            <input name="point_radius_meters" inputMode="numeric" min={1} max={10000} onChange={(event) => setPointRadius(event.target.value)} placeholder="500" value={pointRadius} />
            <span className="field-hint">От 1 до 10000 метров, чтобы геоточка оставалась полезной и безопасной.</span>
          </label>
        </fieldset>
        <fieldset className="editor-fieldset">
          <legend>Маршрут</legend>
          <label>
            Название маршрута
            <input name="route_title" onChange={(event) => setRouteTitle(event.target.value)} placeholder="Утренняя петля" value={routeTitle} />
          </label>
          <label>
            Точки маршрута
            <textarea
              name="route_points"
              onChange={(event) => setRoutePoints(event.target.value)}
              placeholder={"55.751244, 37.618423, старт\n55.761244, 37.628423, финиш"}
              rows={5}
              value={routePoints}
            />
          </label>
        </fieldset>
        <label>
          Отложенная публикация
          <input name="publish_at" onChange={(event) => setPublishAt(event.target.value)} type="datetime-local" value={publishAt} />
          <span className="field-hint">Дата применяется при отправке, если у статьи уже есть право на публикацию.</span>
        </label>
        <div className="editor-actions">
          <button className="primary-button" type="submit">
            Сохранить
          </button>
          <button className="secondary-button" formAction={submitAction} type="submit">
            Отправить
          </button>
          <button className="ghost-button" formAction={archiveAction} type="submit">
            В архив
          </button>
        </div>
      </div>
      <aside className="editor-preview" aria-label="Предпросмотр статьи">
        <span className="eyebrow">Предпросмотр</span>
        <h2>{title || "Заголовок статьи"}</h2>
        <p className="editor-preview-tags">{tags || "теги не указаны"}</p>
        {existingMediaBlocks.length > 0 ? <p className="field-hint">Обложка: первое изображение из медиа-блоков.</p> : null}
        <ArticleDocumentRenderer apiPublicBaseURL={apiPublicBaseURL} content={previewContent} />
      </aside>
    </form>
  );
}

function extractBody(content: ArticleDraft["content"]) {
  const block = content.blocks.find((item) => item.type === "paragraph" && typeof item.text === "string");
  return typeof block?.text === "string" ? block.text : "";
}

function extractMediaBlocks(content: ArticleDraft["content"]) {
  return content.blocks.filter((item): item is ArticleBlock => {
    const block = item as ArticleBlock;
    return block.type === "image" || block.type === "attachment";
  });
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
    radiusMeters: typeof block.radius_meters === "number" ? String(block.radius_meters) : "",
  };
}

function extractRoute(content: ArticleDraft["content"]) {
  const block = content.blocks.find((item) => item.type === "route");
  if (!block || !Array.isArray(block.points)) {
    return null;
  }
  const points = block.points
    .map((point) => {
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

function previewGeoBlocks(pointLabel: string, pointLat: string, pointLng: string, pointRadius: string, routeTitle: string, routePoints: string) {
  const blocks: ArticleDocument["blocks"] = [];
  const pointLatitude = parseCoordinate(pointLat, -90, 90);
  const pointLongitude = parseCoordinate(pointLng, -180, 180);
  if (pointLatitude !== null && pointLongitude !== null) {
    const radiusMeters = parsePositiveNumber(pointRadius);
    blocks.push({
      id: "geo-point-preview",
      type: "geo_point",
      label: pointLabel || "Точка маршрута",
      latitude: pointLatitude,
      longitude: pointLongitude,
      ...(radiusMeters ? { radius_meters: radiusMeters } : {}),
    });
  }
  const points: { latitude: number; longitude: number; label?: string }[] = routePoints
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [lat, lng, label] = line.split(",").map((item) => item.trim());
      const latitude = parseCoordinate(lat, -90, 90);
      const longitude = parseCoordinate(lng, -180, 180);
      if (latitude === null || longitude === null) {
        return null;
      }
      return { latitude, longitude, label: label || undefined };
    })
    .filter((point) => point !== null);
  if (points.length >= 2) {
    blocks.push({ id: "route-preview", type: "route", title: routeTitle || "Маршрут", points });
  }
  return blocks;
}

function parseCoordinate(value: string, min: number, max: number) {
  const parsed = Number(value.replace(",", "."));
  if (!Number.isFinite(parsed) || parsed < min || parsed > max) {
    return null;
  }
  return parsed;
}

function parsePositiveNumber(value: string) {
  const parsed = Number(value.replace(",", "."));
  if (!Number.isFinite(parsed) || parsed <= 0 || parsed > 10000) {
    return null;
  }
  return Math.round(parsed);
}

function formatDateTimeLocal(value: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const offsetMs = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}
