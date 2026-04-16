import { devLogin, requestEmailCode, verifyEmailCode } from "./actions";

type LoginPageProps = {
  searchParams?: Promise<{ email?: string; dev_code?: string; error?: string }>;
};

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const params = await searchParams;
  const error = params?.error;
  const email = params?.email ?? "";

  return (
    <main className="auth-page">
      <section className="auth-panel">
        <span className="brand auth-brand">
          <span className="brand-mark">C</span>
          Catch
        </span>
        <h1>Вход в Catch</h1>
        <p>Введите email, получите короткий код и продолжите с защищённой cookie-сессией.</p>
        {error ? <p className="auth-error">{errorMessage(error)}</p> : null}
        <form action={requestEmailCode} className="auth-form">
          <label>
            Email
            <input name="email" type="email" defaultValue={email} placeholder="you@example.com" required />
          </label>
          <button className="primary-button auth-button" type="submit">
            Получить код
          </button>
        </form>
        <form action={verifyEmailCode} className="auth-form">
          <input name="email" type="hidden" value={email} />
          <label>
            Код
            <input name="code" inputMode="numeric" placeholder={params?.dev_code || "000000"} required />
          </label>
          {params?.dev_code ? <p className="auth-hint">Dev-код: {params.dev_code}</p> : null}
          <button className="primary-button auth-button" type="submit">
            Войти по коду
          </button>
        </form>
        <div className="auth-divider">или</div>
        <form action={devLogin}>
          <button className="primary-button auth-button" type="submit">
            Войти как dev-пользователь
          </button>
        </form>
      </section>
    </main>
  );
}

function errorMessage(error: string) {
  switch (error) {
    case "dev-login-unavailable":
      return "Dev-вход недоступен в текущем окружении.";
    case "email-required":
      return "Укажите email.";
    case "email-request-failed":
      return "Не удалось отправить код.";
    case "code-required":
      return "Укажите код из письма.";
    case "code-invalid":
      return "Код не подошёл или истёк.";
    default:
      return "Не удалось выполнить вход.";
  }
}
