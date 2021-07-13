package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pctl show", func() {
	Context("show", func() {
		It("returns information about the given profile", func() {
			cmd := exec.Command(binaryPath, "show", "nginx-catalog/weaveworks-nginx")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("Catalog      \tnginx-catalog                                      \t\n" +
				"Name         \tweaveworks-nginx                                   \t\n" +
				"Version      \tv0.1.0                                             \t\n" +
				"Description  \tThis installs nginx.                               \t\n" +
				"URL          \thttps://github.com/weaveworks/profiles-examples    \t\n" +
				"Maintainer   \tweaveworks (https://github.com/weaveworks/profiles)\t\n" +
				"Prerequisites\tKubernetes 1.18+                                   \t\n"))
		})

		When("version is used in the catalog", func() {
			It("shows the right profile", func() {
				cmd := exec.Command(binaryPath, "show", "nginx-catalog/weaveworks-nginx/v0.1.0")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("Catalog      \tnginx-catalog                                      \t\n" +
					"Name         \tweaveworks-nginx                                   \t\n" +
					"Version      \tv0.1.0                                             \t\n" +
					"Description  \tThis installs nginx.                               \t\n" +
					"URL          \thttps://github.com/weaveworks/profiles-examples    \t\n" +
					"Maintainer   \tweaveworks (https://github.com/weaveworks/profiles)\t\n" +
					"Prerequisites\tKubernetes 1.18+                                   \t\n"))
			})
		})

		When("-o is set to json", func() {
			It("returns the profile info in json", func() {
				cmd := exec.Command(binaryPath, "show", "-o", "json", "nginx-catalog/weaveworks-nginx")
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(session)).To(ContainSubstring(`{
  "tag": "weaveworks-nginx/v0.1.0",
  "catalogSource": "nginx-catalog",
  "url": "https://github.com/weaveworks/profiles-examples",
  "name": "weaveworks-nginx",
  "description": "This installs nginx.",
  "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
  "prerequisites": [
    "Kubernetes 1.18+"
  ]
}`))
			})
		})

		When("a name argument is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "show")
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("argument must be provided"))
			})
		})
	})
})
