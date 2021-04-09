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
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Show", func() {
	var (
		fakeHTTPClient *fakes.FakeHTTPClient
	)

	BeforeEach(func() {
		fakeHTTPClient = new(fakes.FakeHTTPClient)
		catalog.SetHTTPClient(fakeHTTPClient)
	})

	When("the profile exists in the catalog", func() {
		It("returns all information about the profile", func() {
			httpBody := bytes.NewBufferString(`
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "0.0.1",
	"catalog": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}
		  `)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			resp, err := catalog.Show("http://example.catalog", "weaveworks-nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.DoCallCount()).To(Equal(1))
			req := fakeHTTPClient.DoArgsForCall(0)
			Expect(req.URL.String()).To(Equal("http://example.catalog/profiles/weaveworks-nginx"))
			Expect(resp).To(Equal(
				profilesv1.ProfileDescription{
					Name:          "nginx-1",
					Description:   "nginx 1",
					Version:       "0.0.1",
					Catalog:       "weaveworks (https://github.com/weaveworks/profiles)",
					URL:           "https://github.com/weaveworks/nginx-profile",
					Prerequisites: []string{"Kubernetes 1.18+"},
					Maintainer:    "WeaveWorks <gitops@weave.works>",
				},
			))
		})
	})

	When("the catalog url is invalid", func() {
		It("returns an error", func() {
			_, err := catalog.Show("!\"££!\"£%$£$%%^&&^*()~{@}:@.|ZX", "weaveworks-nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to parse url")))
		})
	})

	When("the profile does not exist in the catalog", func() {
		It("returns an error", func() {
			fakeHTTPClient.DoReturns(&http.Response{
				StatusCode: http.StatusNotFound,
				Body:       ioutil.NopCloser(nil),
			}, nil)

			_, err := catalog.Show("http://example.catalog", "dontexist")
			Expect(err).To(MatchError("unable to find profile `dontexist` in catalog http://example.catalog"))
			req := fakeHTTPClient.DoArgsForCall(0)
			Expect(req.URL.String()).To(Equal("http://example.catalog/profiles/dontexist"))
		})
	})

	When("http request returns any other non-200 code", func() {
		It("returns an error", func() {
			fakeHTTPClient.DoReturns(&http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       ioutil.NopCloser(nil),
			}, nil)
			_, err := catalog.Show("http://example.catalog", "weaveworks-nginx")
			Expect(err).To(MatchError("failed to fetch profile: status code 502"))
		})
	})

	When("http request fails", func() {
		It("returns an error", func() {
			fakeHTTPClient.DoReturns(nil, fmt.Errorf("epic fail"))
			_, err := catalog.Show("http://example.catalog", "weaveworks-nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to do request: epic fail")))
		})
	})

	When("the profile isn't valid json", func() {
		It("returns an error", func() {
			httpBody := bytes.NewBufferString(`!20342 totally n:ot json "`)
			fakeHTTPClient.DoReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			_, err := catalog.Show("http://example.catalog", "weaveworks-nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile")))
		})
	})
})
