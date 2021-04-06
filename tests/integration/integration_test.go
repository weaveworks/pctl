package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("PCTL", func() {
	var exampleCatalog string
	BeforeEach(func() {
		exampleCatalog = "http://localhost:8080"
	})

	Context("search", func() {
		It("returns the matching profiles", func() {
			cmd := exec.Command(binaryPath, "--catalog-url", exampleCatalog, "search", "nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("weaveworks-nginx: This installs nginx"))
		})

		When("catalog-url is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "search", "nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("--catalog-url or $PCTL_CATALOG_URL must be provided"))
			})
		})
	})

	Context("show", func() {
		It("returns information about the given profile", func() {
			cmd := exec.Command(binaryPath, "--catalog-url", exampleCatalog, "show", "weaveworks-nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("retrieving informtation for profile \"weaveworks-nginx\":"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("name: weaveworks-nginx"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("description: This installs nginx."))
			Expect(string(session.Out.Contents())).To(ContainSubstring("version: 0.0.1"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("catalog: weaveworks (https://github.com/weaveworks/profiles)"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("prerequisites:\n- Kubernetes 1.18+"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("maintainer: WeaveWorks <gitops@weave.works>"))
		})

		When("the profile is not listed in the catalog", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "--catalog-url", exampleCatalog, "show", "unlisted")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("unable to find profile `unlisted` in catalog http://localhost:8080"))
			})
		})

		When("catalog-url is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "show", "weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("--catalog-url or $PCTL_CATALOG_URL must be provided"))
			})
		})
	})
})
