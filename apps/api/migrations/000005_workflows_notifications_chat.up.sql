create table if not exists notifications (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    event_type text not null,
    target_type text,
    target_id text,
    title text not null,
    body text not null default '',
    unread_count integer not null default 1,
    read_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint notifications_unread_count_check check (unread_count > 0)
);

create index if not exists notifications_user_unread_idx on notifications(user_id, read_at, updated_at desc);
create index if not exists notifications_group_idx on notifications(user_id, event_type, target_type, target_id);

create table if not exists moderation_submissions (
    id uuid primary key default gen_random_uuid(),
    article_id uuid not null references articles(id) on delete cascade,
    revision_id uuid not null references article_revisions(id) on delete cascade,
    author_id uuid not null references users(id) on delete restrict,
    status text not null default 'pending',
    rejection_reason text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    decided_at timestamptz,
    constraint moderation_submissions_status_check check (status in ('pending', 'approved', 'rejected', 'cancelled'))
);

create unique index if not exists moderation_submissions_revision_unique on moderation_submissions(revision_id);
create index if not exists moderation_submissions_status_idx on moderation_submissions(status, created_at);

create table if not exists moderation_approvals (
    submission_id uuid not null references moderation_submissions(id) on delete cascade,
    moderator_id uuid not null references users(id) on delete restrict,
    is_admin_approval boolean not null default false,
    created_at timestamptz not null default now(),
    primary key (submission_id, moderator_id)
);

create table if not exists moderation_threads (
    id uuid primary key default gen_random_uuid(),
    submission_id uuid not null references moderation_submissions(id) on delete cascade,
    author_id uuid not null references users(id) on delete restrict,
    block_id text,
    body text not null,
    status text not null default 'open',
    created_at timestamptz not null default now(),
    resolved_at timestamptz,
    resolved_by uuid references users(id) on delete restrict,
    constraint moderation_threads_status_check check (status in ('open', 'resolved'))
);

create index if not exists moderation_threads_submission_idx on moderation_threads(submission_id, status);

create table if not exists reports (
    id uuid primary key default gen_random_uuid(),
    target_type text not null,
    target_id uuid not null,
    reporter_id uuid not null references users(id) on delete restrict,
    reason text not null,
    details text,
    status text not null default 'pending',
    created_at timestamptz not null default now(),
    decided_at timestamptz,
    constraint reports_target_type_check check (target_type in ('article', 'comment')),
    constraint reports_reason_check check (reason in ('advertising', 'profanity', 'insult', 'fraud', 'other')),
    constraint reports_status_check check (status in ('pending', 'accepted', 'rejected')),
    constraint reports_other_details_check check (reason <> 'other' or nullif(details, '') is not null)
);

create unique index if not exists reports_unique_reason_idx on reports(target_type, target_id, reporter_id, reason);
create index if not exists reports_status_idx on reports(status, created_at);

create table if not exists report_decisions (
    report_id uuid not null references reports(id) on delete cascade,
    moderator_id uuid not null references users(id) on delete restrict,
    decision text not null,
    is_admin_decision boolean not null default false,
    created_at timestamptz not null default now(),
    primary key (report_id, moderator_id),
    constraint report_decisions_decision_check check (decision in ('accept', 'reject'))
);

create table if not exists chat_conversations (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists chat_conversation_members (
    conversation_id uuid not null references chat_conversations(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    last_read_message_id uuid,
    created_at timestamptz not null default now(),
    primary key (conversation_id, user_id)
);

create table if not exists chat_messages (
    id uuid primary key default gen_random_uuid(),
    conversation_id uuid not null references chat_conversations(id) on delete cascade,
    sender_id uuid not null references users(id) on delete restrict,
    body text not null,
    status text not null default 'sent',
    created_at timestamptz not null default now(),
    read_at timestamptz,
    constraint chat_messages_body_check check (char_length(body) between 1 and 4000),
    constraint chat_messages_status_check check (status in ('sent', 'read'))
);

alter table chat_conversation_members
    add constraint chat_conversation_members_last_read_fk
    foreign key (last_read_message_id) references chat_messages(id) deferrable initially deferred;

create index if not exists chat_messages_conversation_created_idx on chat_messages(conversation_id, created_at, id);
create index if not exists chat_conversation_members_user_idx on chat_conversation_members(user_id);
