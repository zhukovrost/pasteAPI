CREATE TABLE IF NOT EXISTS tokens (
    hash bytea PRIMARY KEY,
    user_id integer REFERENCES users (id) ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL,
    scope varchar(32) NOT NULL
);
