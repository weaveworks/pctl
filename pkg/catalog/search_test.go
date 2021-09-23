package catalog_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/kivo-cli/pkg/catalog"
	"github.com/weaveworks/kivo-cli/pkg/catalog/fakes"
)

var _ = Describe("Search", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
		manager           catalog.Manager
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
	})

	When("profiles matching the search exist", func() {
		It("returns all profiles matching the name description of the profile", func() {
			httpBody := []byte(`{"items":
[
    {
      "name": "nginx-1",
      "description": "nginx 1"
    },
    {
      "name": "nginx-2",
      "description": "nginx 2"
    }
]}
		  `)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			resp, err := manager.Search(fakeCatalogClient, "nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCatalogClient.DoRequestCallCount()).To(Equal(1))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles"))
			Expect(query).To(Equal(map[string]string{
				"name": "nginx",
			}))
			Expect(resp).To(ConsistOf(
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Description: "nginx 1",
					},
					Name: "nginx-1",
				},
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Description: "nginx 2",
					},
					Name: "nginx-2",
				},
			))
		})
	})

	When("user uses search all command", func() {
		It("returns all profiles available from the catalog", func() {
			httpBody := []byte(`{"items":
[
    {
      "name": "nginx-1",
      "description": "nginx 1"
    },
    {
      "name": "nginx-2",
      "description": "nginx 2"
    },
	{
	  "name": "some-new-profile",
	  "description": "some new profile"
	}
]}
		  `)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			resp, err := manager.Search(fakeCatalogClient, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCatalogClient.DoRequestCallCount()).To(Equal(1))
			path, _ := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles"))
			Expect(resp).To(ConsistOf(
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Description: "nginx 1",
					},
					Name: "nginx-1",
				},
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Description: "nginx 2",
					},
					Name: "nginx-2",
				},
				profilesv1.ProfileCatalogEntry{
					ProfileDescription: profilesv1.ProfileDescription{
						Description: "some new profile",
					},
					Name: "some-new-profile",
				},
			))
		})
	})

	When("catalog client fails to make the request", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 502, fmt.Errorf("foo"))
			_, err := manager.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(MatchError("failed to fetch catalog: foo"))
		})
	})

	When("the catalog returns non 200", func() {
		It("returns an error", func() {
			httpBody := []byte(`[]`)
			fakeCatalogClient.DoRequestReturns(httpBody, 500, nil)

			_, err := manager.Search(fakeCatalogClient, "dontexist")
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

			_, err := manager.Search(fakeCatalogClient, "dontexist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to parse catalog")))
		})
	})
})
