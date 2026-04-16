drop table if exists article_revision_tags;
drop table if exists tags;
alter table if exists articles drop constraint if exists articles_published_revision_fk;
alter table if exists articles drop constraint if exists articles_current_revision_fk;
drop table if exists article_revisions;
drop table if exists articles;
