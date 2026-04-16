create table if not exists bookmark_lists (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    name text not null,
    position integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint bookmark_lists_name_check check (char_length(name) between 1 and 80)
);

create unique index if not exists bookmark_lists_user_name_unique on bookmark_lists(user_id, lower(name));
create index if not exists bookmark_lists_user_position_idx on bookmark_lists(user_id, position, created_at);

create table if not exists bookmark_items (
    list_id uuid not null references bookmark_lists(id) on delete cascade,
    article_id uuid not null references articles(id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key (list_id, article_id)
);

create index if not exists bookmark_items_article_idx on bookmark_items(article_id);

create table if not exists follows (
    follower_id uuid not null references users(id) on delete cascade,
    author_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key (follower_id, author_id),
    constraint follows_not_self_check check (follower_id <> author_id)
);

create index if not exists follows_author_idx on follows(author_id);
