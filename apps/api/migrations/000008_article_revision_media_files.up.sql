create table if not exists article_revision_media_files (
    revision_id uuid not null references article_revisions(id) on delete cascade,
    file_id uuid not null references media_files(id) on delete restrict,
    position integer not null,
    created_at timestamptz not null default now(),
    primary key (revision_id, file_id),
    constraint article_revision_media_files_position_check check (position >= 0)
);

create index if not exists article_revision_media_files_file_idx on article_revision_media_files(file_id);
