package catalog_test

import (
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Search", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
	})

	When("profiles matching the search exist", func() {
		It("returns all profiles matching the name description of the profile", func() {
			httpBody := []byte(`
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
			fakeCatalogClient.DoRequestReturns(httpBody, nil)

			resp, err := catalog.Search(fakeCatalogClient, "nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCatalogClient.DoRequestCallCount()).To(Equal(1))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles"))
			Expect(query).To(Equal(map[string]string{
				"name": "nginx",
			}))
			Expect(resp).To(ConsistOf(
				profilesv1.ProfileDescription{
					Name:        "nginx-1",
					Description: "nginx 1",
				},
				profilesv1.ProfileDescription{
					Name:        "nginx-2",
					Description: "nginx 2",
				},
			))
		})
	})

	When("no profiles matching the search exist", func() {
		It("returns an error", func() {
			httpBody := []byte(`[]`)
			fakeCatalogClient.DoRequestReturns(httpBody, nil)

			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(MatchError(`no profiles matching "dontexist" found`))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles"))
			Expect(query).To(Equal(map[string]string{
				"name": "dontexist",
			}))
		})
	})

	When("catalog client fails to process request", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, errors.New("status code 502"))
			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("failed to fetch catalog: status code 502"))
		})
	})

	When("http request fails", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, fmt.Errorf("foo"))
			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(MatchError(ContainSubstring("failed to fetch catalog: foo")))
		})
	})

	When("the catalog isn't valid json", func() {
		It("returns an error", func() {
			httpBody := []byte(`!20342 totally n:ot json "`)
			fakeCatalogClient.DoRequestReturns(httpBody, nil)

			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to parse catalog")))
		})
	})

})
