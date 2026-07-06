package tests

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type MockScrobbleRepo struct {
	RecordedScrobbles []model.Scrobble
	ctx               context.Context
}

func (m *MockScrobbleRepo) RecordScrobble(fileID string, submissionTime time.Time) error {
	user, _ := request.UserFrom(m.ctx)
	m.RecordedScrobbles = append(m.RecordedScrobbles, model.Scrobble{
		MediaFileID:    fileID,
		UserID:         user.ID,
		SubmissionTime: submissionTime,
	})
	return nil
}

func (m *MockScrobbleRepo) Recent(limit, offset int, userIDs []string) ([]model.PlayEntry, error) {
	return nil, nil
}

func (m *MockScrobbleRepo) Top(kind string, since time.Time, limit int, userIDs []string) ([]model.TopEntry, error) {
	return nil, nil
}

func (m *MockScrobbleRepo) ActiveUsers() ([]model.ScrobbleUser, error) {
	return nil, nil
}
