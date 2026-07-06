package tests

import (
	"github.com/navidrome/navidrome/model"
)

type MockExternalReleaseRepo struct {
	Releases    map[string][]model.ExternalRelease
	ArtistInfos map[string]*model.ExternalArtistInfo
}

func (m *MockExternalReleaseRepo) ReplaceArtistReleases(artistID, source string, releases []model.ExternalRelease) error {
	if m.Releases == nil {
		m.Releases = make(map[string][]model.ExternalRelease)
	}
	var kept []model.ExternalRelease
	for _, r := range m.Releases[artistID] {
		if r.Source != source {
			kept = append(kept, r)
		}
	}
	m.Releases[artistID] = append(kept, releases...)
	return nil
}

func (m *MockExternalReleaseRepo) GetByArtist(artistID string) ([]model.ExternalRelease, error) {
	return m.Releases[artistID], nil
}

func (m *MockExternalReleaseRepo) GetArtistInfo(artistID, source string) (*model.ExternalArtistInfo, error) {
	info, ok := m.ArtistInfos[artistID+":"+source]
	if !ok {
		return nil, model.ErrNotFound
	}
	return info, nil
}

func (m *MockExternalReleaseRepo) PutArtistInfo(info *model.ExternalArtistInfo) error {
	if m.ArtistInfos == nil {
		m.ArtistInfos = make(map[string]*model.ExternalArtistInfo)
	}
	m.ArtistInfos[info.ArtistID+":"+info.Source] = info
	return nil
}
