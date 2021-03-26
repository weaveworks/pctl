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

		httpBody := bytes.NewBufferString(`
apiVersion: profilecatalog.fluxcd.io/v1alpha1
kind: ProfileCatalog
metadata:
  name: catalog
spec:
  profiles:
    - name: nginx-1
      url: example.com/nginx-1
      description: "nginx 1"
    - name: 2-nginx
      description: "nginx 2"
    - name: something-else
      description: "something else"`)
		fakeHTTPClient.GetReturns(&http.Response{
			Body:       ioutil.NopCloser(httpBody),
			StatusCode: http.StatusOK,
		}, nil)

	})

	When("profiles matching the serach exist", func() {
		It("returns all profiles matching the name description of the profile", func() {
			resp, err := catalog.Search("example.com/profile-catalog.yaml", "nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("example.com/profile-catalog.yaml"))
			Expect(resp).To(ConsistOf(catalog.ProfileDescription{
				Name:        "nginx-1",
				Description: "nginx 1",
			}, catalog.ProfileDescription{
				Name:        "2-nginx",
				Description: "nginx 2",
			}))
		})
	})

	When("no profiles matching the search exist", func() {
		It("returns all profiles matching the name description of the profile", func() {
			_, err := catalog.Search("example.com/profile-catalog.yaml", "dontexist")
			Expect(err).To(MatchError(`no profiles matching "dontexist" found`))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("example.com/profile-catalog.yaml"))
		})
	})

	When("http request fails", func() {
		It("returns an error", func() {
			fakeHTTPClient.GetReturns(&http.Response{
				StatusCode: http.StatusBadGateway,
			}, nil)
			_, err := catalog.Search("example.com/profile-catalog.yaml", "dontexist")
			Expect(err).To(MatchError("failed to fetch catalog: status code 502"))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("example.com/profile-catalog.yaml"))
		})
	})

	When("http request returns non-200", func() {
		It("returns an error", func() {
			fakeHTTPClient.GetReturns(nil, fmt.Errorf("foo"))
			_, err := catalog.Search("example.com/profile-catalog.yaml", "dontexist")
			Expect(err).To(MatchError("failed to fetch catalog: foo"))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("example.com/profile-catalog.yaml"))
		})
	})

	When("the catalog isn't valid yaml", func() {
		It("returns an error", func() {
			httpBody := bytes.NewBufferString(`!20342 totally n:ot yaml "`)
			fakeHTTPClient.GetReturns(&http.Response{
				Body:       ioutil.NopCloser(httpBody),
				StatusCode: http.StatusOK,
			}, nil)

			_, err := catalog.Search("example.com/profile-catalog.yaml", "dontexist")
			Expect(err).To(MatchError(ContainSubstring("failed to parse catalog")))
			Expect(fakeHTTPClient.GetCallCount()).To(Equal(1))
			Expect(fakeHTTPClient.GetArgsForCall(0)).To(Equal("example.com/profile-catalog.yaml"))
		})
	})
})
