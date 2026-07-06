package tests

import (
	"time"

	"github.com/navidrome/navidrome/model"
)

type MockReleaseAlertRepo struct {
	Alerts []model.ReleaseAlert
}

func (m *MockReleaseAlertRepo) InsertIfMissing(alert model.ReleaseAlert) (bool, error) {
	for _, a := range m.Alerts {
		if a.ID == alert.ID {
			return false, nil
		}
	}
	alert.CreatedAt = time.Now()
	m.Alerts = append(m.Alerts, alert)
	return true, nil
}

func (m *MockReleaseAlertRepo) GetAll(limit int) ([]model.ReleaseAlert, error) {
	if len(m.Alerts) > limit {
		return m.Alerts[:limit], nil
	}
	return m.Alerts, nil
}

func (m *MockReleaseAlertRepo) CountSince(t time.Time) (int64, error) {
	var count int64
	for _, a := range m.Alerts {
		if a.CreatedAt.After(t) {
			count++
		}
	}
	return count, nil
}
