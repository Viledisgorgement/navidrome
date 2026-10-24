package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("ReleaseAlertRepository", func() {
	var repo model.ReleaseAlertRepository
	var db dbx.Builder

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		db = GetDBXBuilder()
		repo = NewReleaseAlertRepository(ctx, db)
	})

	AfterEach(func() {
		_, _ = db.NewQuery("DELETE FROM release_alert").Execute()
	})

	It("sorts by release date newest first, with unparseable dates last", func() {
		for id, date := range map[string]string{
			"a": "2026-09-09",
			"b": "2026-09-25",
			"c": "TBA",
			"d": "2026-10-03",
			"e": "2026-09-18",
			"f": "2026-10",
		} {
			_, err := repo.InsertIfMissing(model.ReleaseAlert{
				ID: id, ArtistID: artistKraftwerk.ID, ArtistName: artistKraftwerk.Name,
				Source: model.ReleaseSourceMetalArchives, Title: id, ReleaseDate: date,
			})
			Expect(err).ToNot(HaveOccurred())
		}

		alerts, err := repo.GetAll(50)
		Expect(err).ToNot(HaveOccurred())
		var dates []string
		for _, a := range alerts {
			dates = append(dates, a.ReleaseDate)
		}
		Expect(dates).To(Equal([]string{"2026-10-03", "2026-10", "2026-09-25", "2026-09-18", "2026-09-09", "TBA"}))
	})
})
