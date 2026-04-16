create table if not exists audit_log (
    id bigserial primary key,
    actor_user_id uuid references users(id) on delete set null,
    method text not null,
    path text not null,
    status integer not null,
    trace_id text,
    ip inet,
    user_agent text,
    created_at timestamptz not null default now()
);

create index if not exists audit_log_actor_created_idx on audit_log(actor_user_id, created_at desc);
create index if not exists audit_log_path_created_idx on audit_log(path, created_at desc);
