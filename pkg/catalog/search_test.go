package catalog_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
)

var _ = Describe("Search", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("profiles matching the search exist", func() {
		It("returns all profiles matching the name description of the profile", func() {
			httpBody := bytes.NewBufferString(`
[
    {
      "name": "nginx-1",
      "description": "nginx 1"
    },
    {
      "name": "nginx-2",
      "description": "nginx 2"
    }
]
		  `)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			resp, err := catalog.Search("http://example.catalog", "nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			req := fakeHTTPClient.DoArgsForCall(0)
			Expect(req.URL.String()).To(Equal("http://example.catalog/profiles?name=nginx"))
			Expect(resp).To(ConsistOf(
				catalog.ProfileDescription{
					Name:        "nginx-1",
					Description: "nginx 1",
				},
				catalog.ProfileDescription{
					Name:        "nginx-2",
					Description: "nginx 2",
				},
			))
		})
	})

	When("no profiles matching the search exist", func() {
		It("returns an error", func() {
			httpBody := bytes.NewBufferString(`[]`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			_, err := catalog.Search("http://example.catalog", "dontexist")
			Expect(err).To(MatchError(`no profiles matching "dontexist" found`))
			req := fakeHTTPClient.DoArgsForCall(0)
			Expect(req.URL.String()).To(Equal("http://example.catalog/profiles?name=dontexist"))
		})
	})

	When("http request returns non-200", func() {
		It("returns an error", func() {
			fakeHTTPClient.DoReturns(&http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       ioutil.NopCloser(nil),
			}, nil)
			_, err := catalog.Search("http://example.catalog", "dontexist")
			Expect(err).To(MatchError("failed to fetch catalog: status code 502"))
		})
	})

	When("http request fails", func() {
		It("returns an error", func() {
			fakeHTTPClient.DoReturns(nil, fmt.Errorf("foo"))
			_, err := catalog.Search("http://example.catalog", "dontexist")
			Expect(err).To(MatchError(ContainSubstring("failed to do request: foo")))
		})
	})

	When("the catalog isn't valid json", func() {
		It("returns an error", func() {
			httpBody := bytes.NewBufferString(`!20342 totally n:ot json "`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			_, err := catalog.Search("http://example.catalog", "dontexist")
			Expect(err).To(MatchError(ContainSubstring("failed to parse catalog")))
		})
	})

	When("the catalog url is invalid", func() {
		It("returns an error", func() {
			_, err := catalog.Search("!\"££!\"£%$£$%%^&&^*()~{@}:@.|ZX", "nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to parse url")))
		})
	})

})
