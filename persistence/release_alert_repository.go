package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// recentReleaseWindow matches how far back the release check looks for new
// releases (core/community newReleaseWindow).
const recentReleaseWindow = 14 * 24 * time.Hour

type releaseAlertRepository struct {
	sqlRepository
}

func NewReleaseAlertRepository(ctx context.Context, db dbx.Builder) model.ReleaseAlertRepository {
	r := &releaseAlertRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "release_alert"
	return r
}

func (r *releaseAlertRepository) InsertIfMissing(alert model.ReleaseAlert) (bool, error) {
	insert := Insert(r.tableName).
		Options("OR IGNORE").
		SetMap(map[string]any{
			"id":           alert.ID,
			"artist_id":    alert.ArtistID,
			"artist_name":  alert.ArtistName,
			"source":       alert.Source,
			"title":        alert.Title,
			"release_type": alert.ReleaseType,
			"release_date": alert.ReleaseDate,
			"external_url": alert.ExternalURL,
			"created_at":   time.Now().Unix(),
		})
	affected, err := r.executeSQL(insert)
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

type dbReleaseAlert struct {
	ID          string `db:"id"`
	ArtistID    string `db:"artist_id"`
	ArtistName  string `db:"artist_name"`
	Source      string `db:"source"`
	Title       string `db:"title"`
	ReleaseType string `db:"release_type"`
	ReleaseDate string `db:"release_date"`
	ExternalURL string `db:"external_url"`
	CreatedAt   int64  `db:"created_at"`
}

func (r *releaseAlertRepository) GetAll(limit int) ([]model.ReleaseAlert, error) {
	// ISO dates (YYYY, YYYY-MM, YYYY-MM-DD) compare correctly as text; a
	// partial date is compared against the cutoff at the same precision.
	// Recent and upcoming releases come first, soonest at the top; older
	// alerts follow, newest first; anything else ("TBA", empty) goes last.
	cutoff := time.Now().Add(-recentReleaseWindow).Format("2006-01-02")
	sq := Select("*").From(r.tableName).
		OrderByClause(`case
			when release_date is null or release_date not glob '[0-9][0-9][0-9][0-9]*' then 2
			when release_date >= substr(?, 1, length(release_date)) then 0
			else 1 end`, cutoff).
		OrderByClause("case when release_date >= substr(?, 1, length(release_date)) then release_date end asc", cutoff).
		OrderBy("release_date desc", "created_at desc").
		Limit(uint64(limit)) //nolint:gosec
	var rows []dbReleaseAlert
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}
	alerts := make([]model.ReleaseAlert, len(rows))
	for i, row := range rows {
		alerts[i] = model.ReleaseAlert{
			ID:          row.ID,
			ArtistID:    row.ArtistID,
			ArtistName:  row.ArtistName,
			Source:      row.Source,
			Title:       row.Title,
			ReleaseType: row.ReleaseType,
			ReleaseDate: row.ReleaseDate,
			ExternalURL: row.ExternalURL,
			CreatedAt:   time.Unix(row.CreatedAt, 0),
		}
	}
	return alerts, nil
}

func (r *releaseAlertRepository) CountSince(t time.Time) (int64, error) {
	return r.count(Select().Where(Gt{"created_at": t.Unix()}))
}
