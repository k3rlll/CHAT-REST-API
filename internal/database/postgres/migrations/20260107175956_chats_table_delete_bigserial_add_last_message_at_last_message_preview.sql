-- +goose Up
-- +goose StatementBegin
ALTER TABLE chats ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ;
ALTER TABLE chats ADD COLUMN IF NOT EXISTS last_message_preview VARCHAR(512);
DROP TABLE IF EXISTS messages;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
ALTER TABLE chats DROP COLUMN IF EXISTS last_message_at;
ALTER TABLE chats DROP COLUMN IF EXISTS last_message_preview;
ALTER TABLE chats ALTER COLUMN id SET DATA TYPE BIGSERIAL;
CREATE TABLE messages(
    id BIGSERIAL PRIMARY KEY,
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    chat_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    sender_username VARCHAR(255) NOT NULL,
    FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd
