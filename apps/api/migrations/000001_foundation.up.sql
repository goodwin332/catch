create extension if not exists pgcrypto;

create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    email text not null,
    username text,
    display_name text,
    avatar_url text,
    rating integer not null default 0,
    role text not null default 'user',
    status text not null default 'active',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint users_rating_max_check check (rating <= 1000000),
    constraint users_role_check check (role in ('user', 'admin')),
    constraint users_status_check check (status in ('active', 'blocked', 'deleted'))
);

create unique index if not exists users_email_lower_unique on users (lower(email));
create unique index if not exists users_username_lower_unique on users (lower(username)) where username is not null;

create table if not exists user_profiles (
    user_id uuid primary key references users(id) on delete cascade,
    birth_date date,
    bio text,
    boat text,
    country_code text,
    country_name text,
    city_name text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists auth_sessions (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    token_hash bytea not null,
    csrf_token_hash bytea not null,
    user_agent text,
    ip inet,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index if not exists auth_sessions_token_hash_unique on auth_sessions(token_hash);
create index if not exists auth_sessions_user_id_idx on auth_sessions(user_id);
create index if not exists auth_sessions_expires_at_idx on auth_sessions(expires_at);

create table if not exists email_login_codes (
    id uuid primary key default gen_random_uuid(),
    email text not null,
    code_hash bytea not null,
    purpose text not null,
    attempts integer not null default 0,
    request_ip inet,
    expires_at timestamptz not null,
    consumed_at timestamptz,
    created_at timestamptz not null default now(),
    constraint email_login_codes_purpose_check check (purpose in ('login', 'registration')),
    constraint email_login_codes_attempts_check check (attempts >= 0)
);

create index if not exists email_login_codes_email_lower_idx on email_login_codes(lower(email));
create index if not exists email_login_codes_expires_at_idx on email_login_codes(expires_at);

create table if not exists oauth_accounts (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    provider text not null,
    provider_account_id text not null,
    email text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint oauth_accounts_provider_check check (provider in ('google', 'vk', 'yandex'))
);

create unique index if not exists oauth_accounts_provider_account_unique on oauth_accounts(provider, provider_account_id);
create index if not exists oauth_accounts_user_id_idx on oauth_accounts(user_id);

create table if not exists outbox_events (
    id bigserial primary key,
    aggregate_type text not null,
    aggregate_id text not null,
    event_type text not null,
    payload jsonb not null,
    status text not null default 'pending',
    attempts integer not null default 0,
    available_at timestamptz not null default now(),
    locked_at timestamptz,
    locked_by text,
    last_error text,
    created_at timestamptz not null default now(),
    processed_at timestamptz,
    constraint outbox_events_status_check check (status in ('pending', 'processing', 'processed', 'failed')),
    constraint outbox_events_attempts_check check (attempts >= 0)
);

create index if not exists outbox_events_pending_idx
    on outbox_events(available_at, id)
    where status = 'pending';

create index if not exists outbox_events_aggregate_idx
    on outbox_events(aggregate_type, aggregate_id);
