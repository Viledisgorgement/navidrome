package artwork

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fromURL", func() {
	It("refuses to fetch artwork from a loopback address", func() {
		hit := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hit = true
			_, _ = w.Write([]byte("secret"))
		}))
		defer srv.Close()

		u, _ := url.Parse(srv.URL + "/cover.jpg")
		r, path, err := fromURL(GinkgoT().Context(), u)
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(BeNil())
		Expect(path).To(BeEmpty())
		Expect(hit).To(BeFalse())
	})

	It("refuses a hostname that resolves to loopback", func() {
		u, _ := url.Parse("http://localhost:1/cover.jpg")
		r, _, err := fromURL(GinkgoT().Context(), u)
		Expect(err).ToNot(HaveOccurred())
		Expect(r).To(BeNil())
	})
})
