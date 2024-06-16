CREATE TABLE IF NOT EXISTS pastes (
    id serial PRIMARY KEY,
    title VARCHAR(255) NOT NULL DEFAULT '',
    category SMALLINT NULL DEFAULT 0,
    text TEXT NOT NULL DEFAULT '', -- TODO: make it post to blob store
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    expires_at timestamp(0) with time zone NOT NULL,
    version INT NOT NULL DEFAULT 1
    -- TODO: make images
    -- TODO: make tags
    -- TODO: make users
);
