create table if not exists articles (
    id uuid primary key default gen_random_uuid(),
    author_id uuid not null references users(id) on delete restrict,
    status text not null default 'draft',
    current_revision_id uuid,
    published_revision_id uuid,
    moderation_required boolean not null default false,
    scheduled_at timestamptz,
    published_at timestamptz,
    removed_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint articles_status_check check (status in ('draft', 'in_moderation', 'ready_to_publish', 'published', 'archived', 'removed'))
);

create index if not exists articles_author_status_idx on articles(author_id, status, updated_at desc);
create index if not exists articles_public_idx on articles(published_at desc, id) where status = 'published';

create table if not exists article_revisions (
    id uuid primary key default gen_random_uuid(),
    article_id uuid not null references articles(id) on delete cascade,
    author_id uuid not null references users(id) on delete restrict,
    version integer not null,
    title text not null,
    content jsonb not null,
    excerpt text not null default '',
    status text not null default 'draft',
    created_at timestamptz not null default now(),
    constraint article_revisions_title_check check (char_length(title) between 3 and 160),
    constraint article_revisions_version_check check (version > 0),
    constraint article_revisions_status_check check (status in ('draft', 'submitted', 'approved', 'rejected', 'published'))
);

create unique index if not exists article_revisions_article_version_unique on article_revisions(article_id, version);
create index if not exists article_revisions_article_created_idx on article_revisions(article_id, created_at desc);

alter table articles
    add constraint articles_current_revision_fk
    foreign key (current_revision_id) references article_revisions(id) deferrable initially deferred;

alter table articles
    add constraint articles_published_revision_fk
    foreign key (published_revision_id) references article_revisions(id) deferrable initially deferred;

create table if not exists tags (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    slug text not null,
    created_at timestamptz not null default now(),
    constraint tags_name_check check (char_length(name) between 1 and 64)
);

create unique index if not exists tags_slug_unique on tags(slug);

create table if not exists article_revision_tags (
    revision_id uuid not null references article_revisions(id) on delete cascade,
    tag_id uuid not null references tags(id) on delete restrict,
    position integer not null,
    primary key (revision_id, tag_id),
    constraint article_revision_tags_position_check check (position >= 0 and position < 10)
);

create index if not exists article_revision_tags_tag_idx on article_revision_tags(tag_id);
