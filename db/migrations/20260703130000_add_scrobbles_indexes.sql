-- +goose Up
-- +goose StatementBegin
CREATE INDEX scrobbles_user_date ON scrobbles(user_id, submission_time);
CREATE INDEX scrobbles_file ON scrobbles(media_file_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS scrobbles_user_date;
DROP INDEX IF EXISTS scrobbles_file;
-- +goose StatementEnd
