package catalog_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
)

var _ = Describe("GetAvailableUpdates", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
	})

	When("profiles are available with higher version than the installed one", func() {
		It("returns all those profiles", func() {
			httpBody := []byte(`{"items":
[
    {
      	"name": "nginx-1",
      	"description": "nginx 1",
     	"tag": "v0.0.2"
    },
    {
      	"name": "nginx-1",
      	"description": "nginx 1",
	  	"tag": "v0.0.3"
    }
]}
		  `)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			resp, err := catalog.GetAvailableUpdates(fakeCatalogClient, "catalog", "weaveworks-nginx", "v0.0.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCatalogClient.DoRequestCallCount()).To(Equal(1))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles/catalog/weaveworks-nginx/v0.0.1/available_updates"))
			Expect(query).To(BeNil())
			Expect(resp).To(ConsistOf(
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Name:        "nginx-1",
						Description: "nginx 1",
					},
					Tag: "v0.0.2",
				},
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Name:        "nginx-1",
						Description: "nginx 1",
					},
					Tag: "v0.0.3",
				},
			))
		})
	})

	When("catalog client fails to make the request", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 502, fmt.Errorf("foo"))
			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(MatchError("failed to fetch catalog: foo"))
		})
	})

	When("the catalog returns non 200", func() {
		It("returns an error", func() {
			httpBody := []byte(`[]`)
			fakeCatalogClient.DoRequestReturns(httpBody, 500, nil)

			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(MatchError("failed to fetch profile from catalog, status code 500"))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles"))
			Expect(query).To(Equal(map[string]string{
				"name": "dontexist",
			}))
		})
	})

	When("the catalog isn't valid json", func() {
		It("returns an error", func() {
			httpBody := []byte(`!20342 totally n:ot json "`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			_, err := catalog.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to parse catalog")))
		})
	})
})
