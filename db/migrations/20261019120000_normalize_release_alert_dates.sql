-- +goose Up
-- +goose StatementBegin
-- Metal Archives alerts used to store display dates ("September 9th, 2026"),
-- which sort wrongly as text next to MusicBrainz ISO dates. Convert them to
-- YYYY-MM-DD, or YYYY-MM when only the month is known. Unrecognized values
-- ("TBA") are left alone.
UPDATE release_alert SET release_date =
    trim(substr(release_date, instr(release_date, ',') + 1)) || '-' ||
    CASE substr(release_date, 1, instr(release_date, ' ') - 1)
        WHEN 'January' THEN '01' WHEN 'February' THEN '02' WHEN 'March' THEN '03'
        WHEN 'April' THEN '04' WHEN 'May' THEN '05' WHEN 'June' THEN '06'
        WHEN 'July' THEN '07' WHEN 'August' THEN '08' WHEN 'September' THEN '09'
        WHEN 'October' THEN '10' WHEN 'November' THEN '11' WHEN 'December' THEN '12'
    END || '-' ||
    printf('%02d', CAST(substr(release_date, instr(release_date, ' ') + 1) AS INTEGER))
WHERE release_date GLOB '[A-Z]* [0-9]*, [0-9][0-9][0-9][0-9]'
  AND substr(release_date, 1, instr(release_date, ' ') - 1) IN
      ('January', 'February', 'March', 'April', 'May', 'June', 'July',
       'August', 'September', 'October', 'November', 'December');

UPDATE release_alert SET release_date =
    substr(release_date, instr(release_date, ' ') + 1) || '-' ||
    CASE substr(release_date, 1, instr(release_date, ' ') - 1)
        WHEN 'January' THEN '01' WHEN 'February' THEN '02' WHEN 'March' THEN '03'
        WHEN 'April' THEN '04' WHEN 'May' THEN '05' WHEN 'June' THEN '06'
        WHEN 'July' THEN '07' WHEN 'August' THEN '08' WHEN 'September' THEN '09'
        WHEN 'October' THEN '10' WHEN 'November' THEN '11' WHEN 'December' THEN '12'
    END
WHERE release_date GLOB '[A-Z]* [0-9][0-9][0-9][0-9]'
  AND substr(release_date, 1, instr(release_date, ' ') - 1) IN
      ('January', 'February', 'March', 'April', 'May', 'June', 'July',
       'August', 'September', 'October', 'November', 'December');
-- +goose StatementEnd

-- +goose Down
-- Not reversible: the original display text isn't kept.
