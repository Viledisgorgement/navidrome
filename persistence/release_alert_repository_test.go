package persistence

import (
	"context"
	"time"

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

	It("lists recent and upcoming releases soonest first, then older ones, then undated", func() {
		day := func(offset int) string { return time.Now().AddDate(0, 0, offset).Format("2006-01-02") }
		dates := map[string]string{
			"a": day(60),  // far future
			"b": day(3),   // upcoming soon
			"c": "TBA",    // unparseable
			"d": day(-5),  // released recently
			"e": day(-40), // older than the recent window
			"f": day(-90), // oldest
			"g": day(20),
			"h": day(-1)[:7], // month-only, current month
		}
		for id, date := range dates {
			_, err := repo.InsertIfMissing(model.ReleaseAlert{
				ID: id, ArtistID: artistKraftwerk.ID, ArtistName: artistKraftwerk.Name,
				Source: model.ReleaseSourceMetalArchives, Title: id, ReleaseDate: date,
			})
			Expect(err).ToNot(HaveOccurred())
		}

		alerts, err := repo.GetAll(50)
		Expect(err).ToNot(HaveOccurred())
		var ids []string
		for _, a := range alerts {
			ids = append(ids, a.ID)
		}
		Expect(ids).To(Equal([]string{"h", "d", "b", "g", "a", "e", "f", "c"}))
	})
})
