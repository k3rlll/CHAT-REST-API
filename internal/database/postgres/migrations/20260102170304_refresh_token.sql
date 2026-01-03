-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_refresh_token ON refresh_tokens(user_id, refresh_token, expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE refresh_tokens;
DROP INDEX IF EXISTS idx_refresh_token;
-- +goose StatementEnd
