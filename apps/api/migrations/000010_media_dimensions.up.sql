alter table media_files
    add column if not exists width integer,
    add column if not exists height integer;

alter table media_files
    add constraint media_files_width_check check (width is null or width > 0),
    add constraint media_files_height_check check (height is null or height > 0);
