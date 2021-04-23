package integration_test

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("PCTL", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			cmd := exec.Command(binaryPath, "search", "nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("weaveworks-nginx: This installs nginx"))
		})

		When("kubeconfig is incorrectly set", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "--kubeconfig=/non-existing/path/kubeconfig", "search", "nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("failed to create config from kubeconfig path"))
			})
		})

		When("a search string is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "search")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("argument must be provided"))
			})
		})
	})

	Context("show", func() {
		It("returns information about the given profile", func() {
			cmd := exec.Command(binaryPath, "show", "nginx-catalog/weaveworks-nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("retrieving information for profile nginx-catalog/weaveworks-nginx"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("name: weaveworks-nginx"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("description: This installs nginx."))
			Expect(string(session.Out.Contents())).To(ContainSubstring("version: 0.0.1"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("catalog: nginx-catalog"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("prerequisites:\n- Kubernetes 1.18+"))
			Expect(string(session.Out.Contents())).To(ContainSubstring("maintainer: weaveworks (https://github.com/weaveworks/profiles)"))
		})

		When("a name argument is not provided", func() {
			It("returns a useful error", func() {
				cmd := exec.Command(binaryPath, "show")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("argument must be provided"))
			})
		})
	})
	Context("install", func() {
		It("generates a ProfileSubscription ready to be applied to a cluster", func() {
			temp, err := ioutil.TempDir("", "pctl_test_install_generate_01")
			Expect(err).ToNot(HaveOccurred())
			filename := filepath.Join(temp, "profile_subscription.yaml")
			cmd := exec.Command(binaryPath, "install", "--out", filename, "nginx-catalog/weaveworks-nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
status: {}
`))
		})

		When("a branch and namespace is provided", func() {
			It("will use that branch and namespace", func() {
				temp, err := ioutil.TempDir("", "pctl_test_install_generate_branch_01")
				Expect(err).ToNot(HaveOccurred())
				filename := filepath.Join(temp, "profile_subscription.yaml")
				cmd := exec.Command(binaryPath, "install", "--out", filename, "--branch", "my_branch", "--namespace", "my-namespace", "nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				content, err := ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: my-namespace
spec:
  branch: my_branch
  profileURL: https://github.com/weaveworks/nginx-profile
status: {}
`))
			})
		})

		When("a config-secret is provided", func() {
			It("will add a valueFrom section to the profile subscription", func() {
				temp, err := ioutil.TempDir("", "pctl_test_install_generate_values_from_01")
				Expect(err).ToNot(HaveOccurred())
				filename := filepath.Join(temp, "profile_subscription.yaml")
				cmd := exec.Command(binaryPath, "install", "--out", filename, "--config-secret", "my-secret", "nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				content, err := ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: default
spec:
  branch: main
  profileURL: https://github.com/weaveworks/nginx-profile
  valuesFrom:
  - kind: ConfigMap
    name: pctl-profile-values
    valuesKey: my-secret
status: {}
`))
			})
		})

	})
})
