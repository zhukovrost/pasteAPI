ALTER TABLE pastes ADD CONSTRAINT pastes_dates_check CHECK (created_at < expires_at);
ALTER TABLE pastes ADD CONSTRAINT pastes_title_check CHECK (TRIM(title) != '');
ALTER TABLE pastes ADD CONSTRAINT pastes_text_check CHECK (TRIM(text) != '');