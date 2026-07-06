-- +goose Up
-- +goose StatementBegin
CREATE TABLE release_alert(
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    artist_id VARCHAR(255) NOT NULL
        REFERENCES artist(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    artist_name VARCHAR NOT NULL,
    source VARCHAR(32) NOT NULL,
    title VARCHAR NOT NULL,
    release_type VARCHAR(32),
    release_date VARCHAR(32),
    external_url VARCHAR,
    created_at INTEGER NOT NULL
);
CREATE INDEX release_alert_created ON release_alert(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE release_alert;
-- +goose StatementEnd
