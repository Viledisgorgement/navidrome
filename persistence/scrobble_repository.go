package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type scrobbleRepository struct {
	sqlRepository
}

func NewScrobbleRepository(ctx context.Context, db dbx.Builder) model.ScrobbleRepository {
	r := &scrobbleRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "scrobbles"
	return r
}

func (r *scrobbleRepository) RecordScrobble(mediaFileID string, submissionTime time.Time) error {
	userID := loggedUser(r.ctx).ID
	values := map[string]any{
		"media_file_id":   mediaFileID,
		"user_id":         userID,
		"submission_time": submissionTime.Unix(),
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)
	return err
}

type dbPlayEntry struct {
	UserID         string `db:"user_id"`
	UserName       string `db:"user_name"`
	MediaFileID    string `db:"media_file_id"`
	Title          string `db:"title"`
	Artist         string `db:"artist"`
	ArtistID       string `db:"artist_id"`
	Album          string `db:"album"`
	AlbumID        string `db:"album_id"`
	SubmissionTime int64  `db:"submission_time"`
}

func (r *scrobbleRepository) Recent(limit, offset int, userIDs []string) ([]model.PlayEntry, error) {
	sq := Select("s.user_id", "u.user_name", "s.media_file_id", "s.submission_time",
		"mf.title", "mf.artist", "mf.artist_id", "mf.album", "mf.album_id").
		From(r.tableName + " s").
		Join("user u on u.id = s.user_id").
		Join("media_file mf on mf.id = s.media_file_id").
		OrderBy("s.submission_time desc").
		Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec
	if len(userIDs) > 0 {
		sq = sq.Where(Eq{"s.user_id": userIDs})
	}
	var rows []dbPlayEntry
	err := r.queryAll(sq, &rows)
	if err != nil {
		return nil, err
	}
	entries := make([]model.PlayEntry, len(rows))
	for i, row := range rows {
		entries[i] = model.PlayEntry{
			UserID:         row.UserID,
			UserName:       row.UserName,
			MediaFileID:    row.MediaFileID,
			Title:          row.Title,
			Artist:         row.Artist,
			ArtistID:       row.ArtistID,
			Album:          row.Album,
			AlbumID:        row.AlbumID,
			SubmissionTime: time.Unix(row.SubmissionTime, 0),
		}
	}
	return entries, nil
}

func (r *scrobbleRepository) Top(kind string, since time.Time, limit int, userIDs []string) ([]model.TopEntry, error) {
	var sq SelectBuilder
	switch kind {
	case model.TopKindAlbum:
		sq = Select("mf.album_id id", "mf.album name", "mf.album_artist artist",
			"mf.album_artist_id artist_id", "mf.album_id album_id", "count(*) play_count").
			GroupBy("mf.album_id")
	case model.TopKindArtist:
		sq = Select("mf.album_artist_id id", "mf.album_artist name", "count(*) play_count").
			GroupBy("mf.album_artist_id")
	default: // model.TopKindSong
		sq = Select("s.media_file_id id", "mf.title name", "mf.artist artist",
			"mf.artist_id artist_id", "mf.album_id album_id", "count(*) play_count").
			GroupBy("s.media_file_id")
	}
	sq = sq.From(r.tableName + " s").
		Join("media_file mf on mf.id = s.media_file_id").
		OrderBy("play_count desc").
		Limit(uint64(limit)) //nolint:gosec
	if !since.IsZero() {
		sq = sq.Where(GtOrEq{"s.submission_time": since.Unix()})
	}
	if len(userIDs) > 0 {
		sq = sq.Where(Eq{"s.user_id": userIDs})
	}
	var rows []struct {
		ID        string `db:"id"`
		Name      string `db:"name"`
		Artist    string `db:"artist"`
		ArtistID  string `db:"artist_id"`
		AlbumID   string `db:"album_id"`
		PlayCount int64  `db:"play_count"`
	}
	err := r.queryAll(sq, &rows)
	if err != nil {
		return nil, err
	}
	entries := make([]model.TopEntry, len(rows))
	for i, row := range rows {
		entries[i] = model.TopEntry(row)
	}
	return entries, nil
}

func (r *scrobbleRepository) ActiveUsers() ([]model.ScrobbleUser, error) {
	sq := Select("distinct u.id", "u.user_name").
		From(r.tableName + " s").
		Join("user u on u.id = s.user_id").
		OrderBy("u.user_name")
	var rows []struct {
		ID       string `db:"id"`
		UserName string `db:"user_name"`
	}
	err := r.queryAll(sq, &rows)
	if err != nil {
		return nil, err
	}
	users := make([]model.ScrobbleUser, len(rows))
	for i, row := range rows {
		users[i] = model.ScrobbleUser(row)
	}
	return users, nil
}
