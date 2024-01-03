create table if not exists cards (
    user_name text not null,
    bank_name text not null,
    number text,
    cv text,
    password text,
    metadata text,
    primary key (user_name, bank_name, number)
);