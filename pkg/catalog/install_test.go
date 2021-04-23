package catalog_test

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	"github.com/weaveworks/pctl/pkg/writer"
)

var _ = Describe("Install", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
	})

	When("there is an existing catalog and user calls install for a profile", func() {
		It("generates a ProfileSubscription ready to be applied to a cluster", func() {
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
			fakeCatalogClient.DoRequestReturns(httpBody, nil)

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
status: {}
`))
		})

		It("generates a ProfileSubscription with config map data if a config map name is defined", func() {
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
			fakeCatalogClient.DoRequestReturns(httpBody, nil)

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				ConfigMap:     "config-secret",
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(buf).NotTo(BeNil())
			Expect(buf.String()).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: mysub-values
    valuesKey: config-secret
status: {}
`))
		})

		It("returns an error when getting the profile fails", func() {
			fakeCatalogClient.DoRequestReturns([]byte(""), fmt.Errorf("foo"))

			var buf bytes.Buffer
			writer := &writer.StringWriter{
				Out: &buf,
			}
			cfg := catalog.InstallConfig{
				Branch:        "main",
				CatalogName:   "nginx",
				CatalogClient: fakeCatalogClient,
				ConfigMap:     "config-secret",
				Namespace:     "default",
				ProfileName:   "profile",
				SubName:       "mysub",
				Writer:        writer,
			}
			err := catalog.Install(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to get profile \"profile\" in catalog \"nginx\":")))
		})
	})
})
