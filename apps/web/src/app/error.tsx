"use client";

export default function ErrorPage({ reset }: Readonly<{ error: Error & { digest?: string }; reset: () => void }>) {
  return (
    <main className="auth-page">
      <section className="auth-panel">
        <span className="brand auth-brand">
          <span className="brand-mark">C</span>
          Catch
        </span>
        <h1>Что-то пошло не так</h1>
        <p>Страница не смогла получить нужные данные. Попробуйте обновить её ещё раз.</p>
        <button className="primary-button auth-button" onClick={reset} type="button">
          Повторить
        </button>
      </section>
    </main>
  );
}
