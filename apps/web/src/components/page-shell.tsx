import { AppHeader } from "./app-header";

export function PageShell({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="shell">
      <AppHeader />
      <main className="single-page">{children}</main>
    </div>
  );
}
