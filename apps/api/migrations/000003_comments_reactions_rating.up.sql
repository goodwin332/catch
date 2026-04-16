create table if not exists comments (
    id uuid primary key default gen_random_uuid(),
    article_id uuid not null references articles(id) on delete cascade,
    author_id uuid not null references users(id) on delete restrict,
    parent_id uuid references comments(id) on delete cascade,
    body text not null,
    status text not null default 'active',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    edited_at timestamptz,
    deleted_at timestamptz,
    constraint comments_body_check check (char_length(body) between 1 and 4000),
    constraint comments_status_check check (status in ('active', 'deleted'))
);

create index if not exists comments_article_created_idx on comments(article_id, created_at, id);
create index if not exists comments_parent_idx on comments(parent_id);
create index if not exists comments_author_idx on comments(author_id, created_at desc);

create table if not exists reactions (
    id uuid primary key default gen_random_uuid(),
    target_type text not null,
    target_id uuid not null,
    user_id uuid not null references users(id) on delete cascade,
    value integer not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint reactions_target_type_check check (target_type in ('article', 'comment')),
    constraint reactions_value_check check (value in (-1, 1))
);

create unique index if not exists reactions_target_user_unique on reactions(target_type, target_id, user_id);
create index if not exists reactions_target_idx on reactions(target_type, target_id);

create table if not exists rating_events (
    id bigserial primary key,
    user_id uuid not null references users(id) on delete cascade,
    source_type text not null,
    source_id text not null,
    delta integer not null,
    reason text not null,
    created_at timestamptz not null default now(),
    constraint rating_events_reason_check check (reason in (
        'article_published',
        'article_like',
        'article_dislike',
        'comment_like',
        'comment_dislike',
        'follow',
        'unfollow',
        'collection_pick',
        'accepted_article_report',
        'accepted_comment_report'
    ))
);

create index if not exists rating_events_user_created_idx on rating_events(user_id, created_at desc);
create index if not exists rating_events_source_idx on rating_events(source_type, source_id);
