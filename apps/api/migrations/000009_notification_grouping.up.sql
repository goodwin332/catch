with grouped as (
    select
        user_id,
        event_type,
        coalesce(target_type, '') as target_type_key,
        coalesce(target_id, '') as target_id_key,
        (array_agg(id order by updated_at desc, created_at desc))[1] as keep_id,
        sum(unread_count) as unread_total,
        max(updated_at) as max_updated_at
    from notifications
    where read_at is null
    group by user_id, event_type, coalesce(target_type, ''), coalesce(target_id, '')
    having count(*) > 1
),
updated as (
    update notifications n
    set unread_count = g.unread_total,
        updated_at = g.max_updated_at
    from grouped g
    where n.id = g.keep_id
    returning n.id
)
delete from notifications n
using grouped g
where n.user_id = g.user_id
    and n.event_type = g.event_type
    and coalesce(n.target_type, '') = g.target_type_key
    and coalesce(n.target_id, '') = g.target_id_key
    and n.read_at is null
    and n.id <> g.keep_id;

create unique index if not exists notifications_unread_group_unique
    on notifications(user_id, event_type, coalesce(target_type, ''), coalesce(target_id, ''))
    where read_at is null;
