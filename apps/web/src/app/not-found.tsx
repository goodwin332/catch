import { PageShell } from "@/components/page-shell";

export default function NotFound() {
  return (
    <PageShell>
      <section className="empty-state">
        <h1>Такой страницы нет</h1>
        <p>Материал мог быть удалён, скрыт модерацией или ссылка устарела.</p>
        <a className="primary-button" href="/">
          Вернуться в ленту
        </a>
      </section>
    </PageShell>
  );
}
