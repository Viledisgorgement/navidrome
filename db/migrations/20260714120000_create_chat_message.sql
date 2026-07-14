-- +goose Up
-- +goose StatementBegin
CREATE TABLE chat_message(
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL
        REFERENCES user(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    message VARCHAR NOT NULL DEFAULT '',
    image VARCHAR NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);
CREATE INDEX chat_message_created ON chat_message(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE chat_message;
-- +goose StatementEnd
