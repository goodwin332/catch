export function EmptyState({ title, text }: Readonly<{ title: string; text: string }>) {
  return (
    <section className="empty-state">
      <h1>{title}</h1>
      <p>{text}</p>
    </section>
  );
}
