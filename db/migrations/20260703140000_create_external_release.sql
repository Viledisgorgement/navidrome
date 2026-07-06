-- +goose Up
-- +goose StatementBegin
CREATE TABLE external_release(
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    artist_id VARCHAR(255) NOT NULL
        REFERENCES artist(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    source VARCHAR(32) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    title VARCHAR NOT NULL,
    release_type VARCHAR(32),
    release_date VARCHAR(32),
    year INTEGER,
    mbz_release_group_id VARCHAR(255),
    external_url VARCHAR,
    fetched_at INTEGER NOT NULL,
    UNIQUE(source, external_id)
);
CREATE INDEX external_release_artist ON external_release(artist_id, source);

CREATE TABLE external_artist_info(
    artist_id VARCHAR(255) NOT NULL
        REFERENCES artist(id)
            ON DELETE CASCADE
            ON UPDATE CASCADE,
    source VARCHAR(32) NOT NULL,
    external_artist_id VARCHAR(255),
    last_fetched_at INTEGER,
    fetch_status VARCHAR(32),
    PRIMARY KEY(artist_id, source)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE external_release;
DROP TABLE external_artist_info;
-- +goose StatementEnd
