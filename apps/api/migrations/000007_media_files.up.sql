create table if not exists media_files (
    id uuid primary key default gen_random_uuid(),
    uploader_id uuid not null references users(id) on delete restrict,
    storage_key text not null,
    original_name text not null,
    mime_type text not null,
    size_bytes bigint not null,
    status text not null default 'ready',
    created_at timestamptz not null default now(),
    constraint media_files_status_check check (status in ('ready', 'deleted')),
    constraint media_files_size_check check (size_bytes > 0)
);

create unique index if not exists media_files_storage_key_unique on media_files(storage_key);
create index if not exists media_files_uploader_created_idx on media_files(uploader_id, created_at desc);
