create table if not exists credentials (
    id serial,
    user_name text not null,
    login text not null,
    password text not null,
    metadata text,
    primary key (user_name, login)
)