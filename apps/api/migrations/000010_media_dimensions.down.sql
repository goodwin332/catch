alter table media_files
    drop constraint if exists media_files_height_check,
    drop constraint if exists media_files_width_check;

alter table media_files
    drop column if exists height,
    drop column if exists width;
