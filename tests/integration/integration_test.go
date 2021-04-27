package integration_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	repositoryNameTemplate = "https://%s@github.com/weaveworks/pctl-test-repo.git"
)

var _ = Describe("PCTL", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			cmd := exec.Command(binaryPath, "search", "nginx")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("CATALOG/PROFILE               	VERSION	DESCRIPTION                     \n" +
				"nginx-catalog/weaveworks-nginx	0.0.1  	This installs nginx.           \t\n" +
				"nginx-catalog/some-other-nginx	       	This installs some other nginx.\t\n"),
			)
		})

		When("-o is set to json", func() {
			It("returns the matching profiles in json", func() {
				cmd := exec.Command(binaryPath, "search", "-o", "json", "nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(`[
  {
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "0.0.1",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/nginx-profile",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "some-other-nginx",
    "description": "This installs some other nginx.",
    "catalog": "nginx-catalog"
  }
]`))
			})
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
			Expect(string(session.Out.Contents())).To(ContainSubstring("Catalog      \tnginx-catalog                                      \t\n" +
				"Name         \tweaveworks-nginx                                   \t\n" +
				"Version      \t0.0.1                                              \t\n" +
				"Description  \tThis installs nginx.                               \t\n" +
				"URL          \thttps://github.com/weaveworks/nginx-profile        \t\n" +
				"Maintainer   \tweaveworks (https://github.com/weaveworks/profiles)\t\n" +
				"Prerequisites\tKubernetes 1.18+                                   \t\n"))
		})

		When("-o is set to json", func() {
			It("returns the profile info in json", func() {
				cmd := exec.Command(binaryPath, "show", "-o", "json", "nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(string(session.Out.Contents())).To(ContainSubstring(`{
  "name": "weaveworks-nginx",
  "description": "This installs nginx.",
  "version": "0.0.1",
  "catalog": "nginx-catalog",
  "url": "https://github.com/weaveworks/nginx-profile",
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
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("argument must be provided"))
			})
		})
	})
	Context("get", func() {
		var (
			namespace        = "default"
			subscriptionName = "failed-sub"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileSubscription
		)

		BeforeEach(func() {
			profileURL := "https://github.com/weaveworks/nginx-profile"
			pSub = profilesv1.ProfileSubscription{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileSubscription",
					APIVersion: "profilesubscriptions.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      subscriptionName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: profileURL,
					Branch:     "invalid-artifact",
				},
			}
			Expect(kClient.Create(ctx, &pSub)).Should(Succeed())

			profile := profilesv1.ProfileSubscription{}
			Eventually(func() bool {
				err := kClient.Get(ctx, client.ObjectKey{Name: subscriptionName, Namespace: namespace}, &profile)
				return err == nil && len(profile.Status.Conditions) > 0
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			Expect(profile.Status.Conditions[0].Message).To(Equal("error when reconciling profile artifacts"))
			Expect(profile.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(profile.Status.Conditions[0].Status).To(Equal(metav1.ConditionStatus("False")))
			Expect(profile.Status.Conditions[0].Reason).To(Equal("CreateFailed"))
		})

		AfterEach(func() {
			Expect(kClient.Delete(ctx, &pSub)).Should(Succeed())
		})

		It("returns the subscrptions", func() {

			getCmd := func() string {
				cmd := exec.Command(binaryPath, "get", "--namespace", namespace, "--name", subscriptionName)
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				return string(session.Out.Contents())
			}
			Eventually(getCmd).Should(ContainSubstring(`Subscription: failed-sub
Namespace: default
Ready: False
Reason:
 - CreateFailed`))
		})
	})

	Context("list", func() {
		var (
			namespace        = "default"
			subscriptionName = "failed-sub"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileSubscription
		)

		BeforeEach(func() {
			profileURL := "https://github.com/weaveworks/nginx-profile"
			pSub = profilesv1.ProfileSubscription{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileSubscription",
					APIVersion: "profilesubscriptions.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      subscriptionName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileSubscriptionSpec{
					ProfileURL: profileURL,
					Branch:     "invalid-artifact",
				},
			}
			Expect(kClient.Create(ctx, &pSub)).Should(Succeed())

			profile := profilesv1.ProfileSubscription{}
			Eventually(func() bool {
				err := kClient.Get(ctx, client.ObjectKey{Name: subscriptionName, Namespace: namespace}, &profile)
				return err == nil && len(profile.Status.Conditions) > 0
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			Expect(profile.Status.Conditions[0].Message).To(Equal("error when reconciling profile artifacts"))
			Expect(profile.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(profile.Status.Conditions[0].Status).To(Equal(metav1.ConditionStatus("False")))
			Expect(profile.Status.Conditions[0].Reason).To(Equal("CreateFailed"))
		})

		AfterEach(func() {
			Expect(kClient.Delete(ctx, &pSub)).Should(Succeed())
		})

		It("returns the subscrptions", func() {

			listCmd := func() string {
				cmd := exec.Command(binaryPath, "list")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				return string(session.Out.Contents())
			}
			Eventually(listCmd).Should(ContainSubstring(`NAMESPACE	NAME		READY
default		failed-sub	False`))
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

		// Note, the repo cleans the creates PRs via Github actions.
		When("create-pr is enabled", func() {
			It("creates a pull request to the remote branch", func() {
				if _, ok := os.LookupEnv("GIT_TOKEN"); !ok {
					Skip("GIT_TOKEN not set, skipping...")
				}
				temp, err := ioutil.TempDir("", "pctl_test_install_create_pr_01")
				Expect(err).NotTo(HaveOccurred())
				repoLocation := filepath.Join(temp, "repo")
				// clone
				token := os.Getenv("GIT_TOKEN")
				cloneWithToken := fmt.Sprintf(repositoryNameTemplate, token)
				cmd := exec.Command("git", "clone", cloneWithToken, repoLocation)
				err = cmd.Run()
				Expect(err).NotTo(HaveOccurred())
				filename := filepath.Join(repoLocation, "profile_subscription.yaml")
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd = exec.Command(binaryPath,
					"install",
					"--out",
					filename,
					"--create-pr",
					"--branch",
					branch,
					"--repo",
					"weaveworks/pctl-test-repo",
					"nginx-catalog/weaveworks-nginx")
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("PR created with number:"))
			})

			It("fails if repo is not defined", func() {
				temp, err := ioutil.TempDir("", "pctl_test_install_create_pr_02")
				Expect(err).NotTo(HaveOccurred())
				filename := filepath.Join(temp, "profile_subscription.yaml")
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd := exec.Command(binaryPath,
					"install",
					"--out",
					filename,
					"--create-pr",
					"--branch",
					branch,
					"nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("repo must be defined if create-pr is true"))
			})

			It("fails if target location is not a git repository", func() {
				if _, ok := os.LookupEnv("GIT_TOKEN"); !ok {
					// Set up a dummy token, because the SCM client is created before we check the git repo.
					err := os.Setenv("GIT_TOKEN", "dummy")
					Expect(err).ToNot(HaveOccurred())
				}
				temp, err := ioutil.TempDir("", "pctl_test_install_create_pr_03")
				Expect(err).NotTo(HaveOccurred())
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				filename := filepath.Join(temp, "profile_subscription.yaml")
				branch := "prtest_" + suffix
				cmd := exec.Command(binaryPath,
					"install",
					"--out",
					filename,
					"--create-pr",
					"--branch",
					branch,
					"--repo",
					"doesnt/matter",
					"nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("directory is not a git repository"))
			})

			It("fails if target location for install is a folder rather than a file", func() {
				temp, err := ioutil.TempDir("", "pctl_test_install_create_pr_04")
				Expect(err).NotTo(HaveOccurred())
				Expect(err).NotTo(HaveOccurred())
				cmd := exec.Command(binaryPath,
					"install",
					"--out",
					temp,
					"nginx-catalog/weaveworks-nginx")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("is a directory"))
			})
		})
	})
})

func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
