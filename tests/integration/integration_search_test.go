package integration_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pctl search", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			cmd := exec.Command(binaryPath, "search", "nginx")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			expected := "CATALOG/PROFILE               	VERSION	DESCRIPTION                     \n" +
				"nginx-catalog/weaveworks-nginx	v0.1.0 	This installs nginx.           \t\n" +
				"nginx-catalog/weaveworks-nginx	v0.1.1 	This installs nginx.           \t\n" +
				"nginx-catalog/bitnami-nginx   	v0.0.1 	This installs nginx.           \t\n" +
				"nginx-catalog/nginx           	v2.0.1 	This installs nginx.           \t\n" +
				"nginx-catalog/some-other-nginx	       	This installs some other nginx.\t\n\n"
			Expect(string(session)).To(ContainSubstring(expected))
		})

		When("-o is set to json", func() {
			It("returns the matching profiles in json", func() {
				cmd := exec.Command(binaryPath, "search", "-o", "json", "nginx")
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
  },
  {
    "tag": "weaveworks-nginx/v0.1.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "bitnami-nginx/v0.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "bitnami-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "v2.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/nginx-profile",
    "name": "nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "catalogSource": "nginx-catalog",
    "name": "some-other-nginx",
    "description": "This installs some other nginx."
  }`))
			})
		})

		When("kubeconfig is incorrectly set", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "--kubeconfig=/non-existing/path/kubeconfig", "search", "nginx")
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("failed to create config from kubeconfig path"))
			})
		})

		When("a search string is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "search")
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("argument must be provided"))
			})
		})

		When("a search string is provided with all option", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "search", "-all", "nginx")
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("argument must not be provided"))
			})
		})

		It("returns all the profiles with search all option", func() {
			cmd := exec.Command(binaryPath, "search", "--all")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			expected := "CATALOG/PROFILE               	VERSION	DESCRIPTION                     \n" +
				"nginx-catalog/weaveworks-nginx	v0.1.0 	This installs nginx.           \t\n" +
				"nginx-catalog/weaveworks-nginx	v0.1.1 	This installs nginx.           \t\n" +
				"nginx-catalog/bitnami-nginx   	v0.0.1 	This installs nginx.           \t\n" +
				"nginx-catalog/nginx           	v2.0.1 	This installs nginx.           \t\n" +
				"nginx-catalog/some-other-nginx	       	This installs some other nginx.\t\n\n"
			Expect(string(session)).To(ContainSubstring(expected))
		})

		When("-o is set to json with search all", func() {
			It("returns the matching profiles in json", func() {
				cmd := exec.Command(binaryPath, "search", "-a", "-o", "json")
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
  },
  {
    "tag": "weaveworks-nginx/v0.1.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "bitnami-nginx/v0.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "bitnami-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "v2.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/nginx-profile",
    "name": "nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "catalogSource": "nginx-catalog",
    "name": "some-other-nginx",
    "description": "This installs some other nginx."
  }`))
			})
		})
	})
})
