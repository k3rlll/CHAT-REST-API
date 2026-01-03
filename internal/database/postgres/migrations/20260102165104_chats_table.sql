-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE chats (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(20),
    is_private BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TABLE chats;
-- +goose StatementEnd
