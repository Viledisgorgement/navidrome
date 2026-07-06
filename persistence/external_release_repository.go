package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type externalReleaseRepository struct {
	sqlRepository
}

func NewExternalReleaseRepository(ctx context.Context, db dbx.Builder) model.ExternalReleaseRepository {
	r := &externalReleaseRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "external_release"
	return r
}

type dbExternalRelease struct {
	ID                string `db:"id"`
	ArtistID          string `db:"artist_id"`
	Source            string `db:"source"`
	ExternalID        string `db:"external_id"`
	Title             string `db:"title"`
	ReleaseType       string `db:"release_type"`
	ReleaseDate       string `db:"release_date"`
	Year              int    `db:"year"`
	MbzReleaseGroupID string `db:"mbz_release_group_id"`
	ExternalURL       string `db:"external_url"`
	FetchedAt         int64  `db:"fetched_at"`
}

func (r *externalReleaseRepository) ReplaceArtistReleases(artistID, source string, releases []model.ExternalRelease) error {
	del := Delete(r.tableName).Where(And{Eq{"artist_id": artistID}, Eq{"source": source}})
	if _, err := r.executeSQL(del); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, release := range releases {
		insert := Insert(r.tableName).SetMap(map[string]any{
			"id":                   release.ID,
			"artist_id":            artistID,
			"source":               source,
			"external_id":          release.ExternalID,
			"title":                release.Title,
			"release_type":         release.ReleaseType,
			"release_date":         release.ReleaseDate,
			"year":                 release.Year,
			"mbz_release_group_id": release.MbzReleaseGroupID,
			"external_url":         release.ExternalURL,
			"fetched_at":           now,
		})
		if _, err := r.executeSQL(insert); err != nil {
			return err
		}
	}
	return nil
}

func (r *externalReleaseRepository) GetByArtist(artistID string) ([]model.ExternalRelease, error) {
	sq := Select("*").From(r.tableName).
		Where(Eq{"artist_id": artistID}).
		OrderBy("year", "title")
	var rows []dbExternalRelease
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}
	releases := make([]model.ExternalRelease, len(rows))
	for i, row := range rows {
		releases[i] = model.ExternalRelease{
			ID:                row.ID,
			ArtistID:          row.ArtistID,
			Source:            row.Source,
			ExternalID:        row.ExternalID,
			Title:             row.Title,
			ReleaseType:       row.ReleaseType,
			ReleaseDate:       row.ReleaseDate,
			Year:              row.Year,
			MbzReleaseGroupID: row.MbzReleaseGroupID,
			ExternalURL:       row.ExternalURL,
			FetchedAt:         time.Unix(row.FetchedAt, 0),
		}
	}
	return releases, nil
}

func (r *externalReleaseRepository) GetArtistInfo(artistID, source string) (*model.ExternalArtistInfo, error) {
	sq := Select("*").From("external_artist_info").
		Where(And{Eq{"artist_id": artistID}, Eq{"source": source}})
	var row struct {
		ArtistID         string `db:"artist_id"`
		Source           string `db:"source"`
		ExternalArtistID string `db:"external_artist_id"`
		LastFetchedAt    int64  `db:"last_fetched_at"`
		FetchStatus      string `db:"fetch_status"`
	}
	err := r.queryOne(sq, &row)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &model.ExternalArtistInfo{
		ArtistID:         row.ArtistID,
		Source:           row.Source,
		ExternalArtistID: row.ExternalArtistID,
		LastFetchedAt:    time.Unix(row.LastFetchedAt, 0),
		FetchStatus:      row.FetchStatus,
	}, nil
}

func (r *externalReleaseRepository) PutArtistInfo(info *model.ExternalArtistInfo) error {
	del := Delete("external_artist_info").
		Where(And{Eq{"artist_id": info.ArtistID}, Eq{"source": info.Source}})
	if _, err := r.executeSQL(del); err != nil {
		return err
	}
	insert := Insert("external_artist_info").SetMap(map[string]any{
		"artist_id":          info.ArtistID,
		"source":             info.Source,
		"external_artist_id": info.ExternalArtistID,
		"last_fetched_at":    info.LastFetchedAt.Unix(),
		"fetch_status":       info.FetchStatus,
	})
	_, err := r.executeSQL(insert)
	return err
}
