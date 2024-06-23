CREATE TABLE IF NOT EXISTS write_permissions (
    paste_id integer not null references pastes on delete cascade,
    user_id integer not null references users on delete cascade,
    PRIMARY KEY (paste_id, user_id)
);
