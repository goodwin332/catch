import type { ArticleListItem } from "./api";

export const fallbackArticles: ArticleListItem[] = [
  {
    id: "demo-echo",
    author_id: "demo-author-1",
    title: "Как выбрать эхолот для большой воды",
    excerpt: "Короткий разбор датчиков, частот, картографии и настроек, которые помогают не гадать по ряби.",
    tags: ["Эхолот", "Лодка", "Практика"],
    reactions_up: 18,
    reactions_down: 1,
    reaction_score: 17,
    published_at: "2026-04-15T08:30:00Z",
  },
  {
    id: "demo-route",
    author_id: "demo-author-2",
    title: "Маршрут выходного дня: тихая протока и две стоянки",
    excerpt: "Что взять с собой, где безопасно выйти на берег и как не превратить прогулку в марш-бросок.",
    tags: ["Маршрут", "Сплав", "Семья"],
    reactions_up: 11,
    reactions_down: 0,
    reaction_score: 11,
    published_at: "2026-04-14T14:15:00Z",
  },
];

export const articleImages = [
  "https://images.unsplash.com/photo-1478131143081-80f7f84ca84d?auto=format&fit=crop&w=1200&q=80",
  "https://images.unsplash.com/photo-1500530855697-b586d89ba3ee?auto=format&fit=crop&w=1200&q=80",
];
