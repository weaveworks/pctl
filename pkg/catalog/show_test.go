package catalog_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Show", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
	})

	When("the profile exists in the catalog", func() {
		It("returns all information about the profile", func() {
			httpBody := []byte(`
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
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			resp, err := catalog.Show(fakeCatalogClient, "foo", "weaveworks-nginx")
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeCatalogClient.DoRequestCallCount()).To(Equal(1))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles/foo/weaveworks-nginx"))
			Expect(query).To(BeEmpty())
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

	When("http request fails", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 0, errors.New("epic fail"))
			_, err := catalog.Show(fakeCatalogClient, "foo", "weaveworks-nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to do request: epic fail")))
		})
	})

	When("the catalog returns 404", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 404, nil)

			_, err := catalog.Show(fakeCatalogClient, "foo", "dontexist")
			Expect(err).To(MatchError("unable to find profile \"dontexist\" in catalog \"foo\""))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles/foo/dontexist"))
			Expect(query).To(BeEmpty())
		})
	})

	When("the catalog returns a non 200 status code", func() {
		It("returns an error", func() {
			fakeCatalogClient.DoRequestReturns(nil, 500, nil)

			_, err := catalog.Show(fakeCatalogClient, "foo", "dontexist")
			Expect(err).To(MatchError("failed to fetch profile from catalog, status code 500"))
			path, query := fakeCatalogClient.DoRequestArgsForCall(0)
			Expect(path).To(Equal("/profiles/foo/dontexist"))
			Expect(query).To(BeEmpty())
		})
	})

	When("the profile isn't valid json", func() {
		It("returns an error", func() {
			httpBody := []byte(`!20342 totally n:ot json "`)
			fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)

			_, err := catalog.Show(fakeCatalogClient, "foo", "weaveworks-nginx")
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile")))
		})
	})
})
