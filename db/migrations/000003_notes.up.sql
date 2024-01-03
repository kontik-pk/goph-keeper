create table if not exists notes (
    user_name text not null,
    title text not null,
    content text,
    metadata text,
    primary key (user_name, title)
);