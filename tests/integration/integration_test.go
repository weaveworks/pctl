package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("PCTL", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			exampleCatalog := "https://gist.githubusercontent.com/bigkevmcd/dd211661f9b01fa42eade2737f5dc059/raw/8edc1f353bad00a55da009d1834e8455b2e3312f/testing.yaml"
			cmd := exec.Command(binaryPath, "search", "--catalog-url", exampleCatalog, "nginx")
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
				Expect(string(session.Err.Contents())).To(ContainSubstring("--catalog-url must be provided"))
			})
		})
	})
})
