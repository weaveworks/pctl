package integration_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

var pctlTestRepositoryName = "git@github.com:weaveworks/pctl-test-repo.git"

var _ = Describe("PCTL", func() {
	Context("search", func() {
		It("returns the matching profiles", func() {
			cmd := exec.Command(binaryPath, "search", "nginx")
			session, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			expected := "CATALOG/PROFILE               	VERSION	DESCRIPTION                     \n" +
				"nginx-catalog/weaveworks-nginx	v0.1.0 	This installs nginx.           \t\n" +
				"nginx-catalog/some-other-nginx	       	This installs some other nginx.\t\n\n"
			Expect(string(session)).To(ContainSubstring(expected))
		})

		When("-o is set to json", func() {
			It("returns the matching profiles in json", func() {
				cmd := exec.Command(binaryPath, "search", "-o", "json", "nginx")
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(session)).To(ContainSubstring(`{
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "version": "v0.1.0",
    "catalog": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "name": "some-other-nginx",
    "description": "This installs some other nginx.",
    "catalog": "nginx-catalog"
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
	})

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
  "name": "weaveworks-nginx",
  "description": "This installs nginx.",
  "version": "v0.1.0",
  "catalog": "nginx-catalog",
  "url": "https://github.com/weaveworks/profiles-examples",
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

	Context("list", func() {
		var (
			namespace        = "default"
			subscriptionName = "my-sub"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileSubscription
		)

		BeforeEach(func() {
			profileURL := "https://github.com/weaveworks/profiles-examples"
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
					Version:    "weaveworks-nginx/v0.1.0",
					ProfileCatalogDescription: &profilesv1.ProfileCatalogDescription{
						Catalog: "foo",
						Profile: "bar",
						Version: "v0.1.0",
					},
				},
			}
			Expect(kClient.Create(ctx, &pSub)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(kClient.Delete(ctx, &pSub)).Should(Succeed())
		})

		It("returns the subscriptions", func() {
			listCmd := func() []string {
				cmd := exec.Command(binaryPath, "list")
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				return strings.Split(string(session), "\n")
			}
			Eventually(listCmd).Should(ContainElements(
				"NAMESPACE	NAME  	SOURCE         ",
				"default  \tmy-sub\tfoo/bar/v0.1.0\t",
			))
		})
	})

	Context("install", func() {
		var (
			temp      string
			namespace string
		)

		BeforeEach(func() {
			var err error
			namespace = uuid.New().String()
			temp, err = ioutil.TempDir("", "pctl_test_install_generate_branch_01")
			Expect(err).ToNot(HaveOccurred())
			nsp := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(kClient.Create(context.Background(), &nsp)).To(Succeed())
		})

		AfterEach(func() {
			_ = os.RemoveAll(temp)
			nsp := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = kClient.Delete(context.Background(), &nsp)
		})

		It("generates valid artifacts to the local directory", func() {
			subName := "pctl-profile"
			cmd := exec.Command(binaryPath, "install", "--namespace", namespace, "nginx-catalog/weaveworks-nginx/v0.1.0")
			cmd.Dir = temp
			session, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println("Output from failing command: ", string(session))
			}
			Expect(err).ToNot(HaveOccurred())

			var files []string
			profilesDir := filepath.Join(temp, "weaveworks-nginx")
			err = filepath.Walk(profilesDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, profilesDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating the artifacts")
			Expect(files).To(ContainElements(
				"profile.yaml",
				"artifacts/nested-profile/nginx-server/GitRepository.yaml",
				"artifacts/nested-profile/nginx-server/HelmRelease.yaml",
				"artifacts/nginx-deployment/GitRepository.yaml",
				"artifacts/nginx-deployment/Kustomization.yaml",
				"artifacts/dokuwiki/HelmRelease.yaml",
				"artifacts/dokuwiki/HelmRepository.yaml",
			))

			filename := filepath.Join(temp, "weaveworks-nginx", "profile.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  profile_catalog_description:
    catalog: nginx-catalog
    profile: weaveworks-nginx
    version: v0.1.0
  profileURL: https://github.com/weaveworks/profiles-examples
  version: weaveworks-nginx/v0.1.0
status: {}
`, namespace)))

			By("the artifacts being deployable")

			cmd = exec.Command("kubectl", "apply", "-R", "-f", profilesDir)
			cmd.Dir = temp
			session, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println("Output from failing command: ", string(session))
			}
			Expect(err).ToNot(HaveOccurred())

			By("successfully deploying the helm release")
			helmReleaseName := fmt.Sprintf("%s-%s-%s", subName, "bitnami-nginx", "nginx-server")
			var helmRelease *helmv2.HelmRelease
			Eventually(func() bool {
				helmRelease = &helmv2.HelmRelease{}
				err := kClient.Get(context.Background(), client.ObjectKey{Name: helmReleaseName, Namespace: namespace}, helmRelease)
				if err != nil {
					return false
				}
				for _, condition := range helmRelease.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						return true
					}
				}
				return false
			}, 3*time.Minute, 5*time.Second).Should(BeTrue())

			helmOpts := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{"app.kubernetes.io/name": "nginx"},
			}
			var podList *v1.PodList
			Eventually(func() v1.PodPhase {
				podList = &v1.PodList{}
				err := kClient.List(context.Background(), podList, helmOpts...)
				Expect(err).NotTo(HaveOccurred())
				if len(podList.Items) == 0 {
					return v1.PodPhase("")
				}
				return podList.Items[0].Status.Phase
			}, 2*time.Minute, 5*time.Second).Should(Equal(v1.PodPhase("Running")))

			Expect(podList.Items[0].Spec.Containers[0].Image).To(Equal("docker.io/bitnami/nginx:1.19.8-debian-10-r0"))

			By("successfully deploying the kustomize resource")
			kustomizeName := fmt.Sprintf("%s-%s-%s", subName, "weaveworks-nginx", "nginx-deployment")
			var kustomize *kustomizev1.Kustomization
			Eventually(func() bool {
				kustomize = &kustomizev1.Kustomization{}
				err := kClient.Get(context.Background(), client.ObjectKey{Name: kustomizeName, Namespace: namespace}, kustomize)
				if err != nil {
					return false
				}
				for _, condition := range kustomize.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						return true
					}
				}
				return false
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			kustomizeOpts := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{"app": "nginx"},
			}
			Eventually(func() v1.PodPhase {
				podList = &v1.PodList{}
				err := kClient.List(context.Background(), podList, kustomizeOpts...)
				Expect(err).NotTo(HaveOccurred())
				if len(podList.Items) == 0 {
					return v1.PodPhase("no pods found")
				}
				return podList.Items[0].Status.Phase
			}, 2*time.Minute, 5*time.Second).Should(Equal(v1.PodPhase("Running")))

			Expect(podList.Items[0].Spec.Containers[0].Image).To(Equal("nginx:1.14.2"))
		})

		When("a url is provided with a branch and path", func() {
			It("will fetch information from that branch with path", func() {
				namespace := uuid.New().String()
				//subName := "pctl-profile"
				branch := "branch-and-url"
				path := "branch-nginx"
				cmd := exec.Command(binaryPath, "install", "--namespace", namespace, "--profile-url", "https://github.com/weaveworks/profiles-examples", "--profile-branch", branch, "--profile-path", path)
				cmd.Dir = temp
				session, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Println("Output from failing command: ", string(session))
				}
				Expect(err).ToNot(HaveOccurred())

				var files []string
				err = filepath.Walk(temp, func(path string, info os.FileInfo, err error) error {
					files = append(files, path)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				By("creating the artifacts")
				profilesDirProfile := filepath.Join(temp, "profile.yaml")
				profilesArtifacts := filepath.Join(temp, "artifacts")
				profilesArtifactsDeployment := filepath.Join(temp, "artifacts", "nginx-deployment")
				profilesArtifactsDeploymentGitRepo := filepath.Join(temp, "artifacts", "nginx-deployment", "GitRepository.yaml")
				profilesArtifactsDeploymentKustomization := filepath.Join(temp, "artifacts", "nginx-deployment", "Kustomization.yaml")
				Expect(files).To(ContainElements(
					temp,
					profilesDirProfile,
					profilesArtifacts,
					profilesArtifactsDeployment,
					profilesArtifactsDeploymentKustomization,
					profilesArtifactsDeploymentGitRepo,
				))
				filename := filepath.Join(temp, "profile.yaml")
				content, err := ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  branch: branch-and-url
  path: branch-nginx
  profileURL: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace)))
			})
		})

		When("url and catalog entry install format are both defined", func() {
			It("will throw a meaningful error", func() {
				namespace := uuid.New().String()
				//subName := "pctl-profile"
				branch := "branch-and-url"
				path := "branch-nginx"
				cmd := exec.Command(binaryPath, "install", "--namespace", namespace, "--profile-url", "https://github.com/weaveworks/profiles-examples", "--profile-branch", branch, "--profile-path", path, "catalog/profile/v0.0.1")
				cmd.Dir = temp
				session, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Println("Output from failing command: ", string(session))
				}
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("it looks like you provided a url with a catalog entry; please choose either format: url/branch/path or <CATALOG>/<PROFILE>[/<VERSION>]"))
			})
		})

		When("a catalog version is provided, but it's an invalid/missing version", func() {
			It("provide an error saying the profile with these specifics can't be found", func() {
				cmd := exec.Command(binaryPath, "install", "nginx-catalog/weaveworks-nginx/v999.9.9")
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring(`unable to find profile "weaveworks-nginx" in catalog "nginx-catalog" (with version if provided: v999.9.9)`))
			})
		})

		// Note, the repo cleans the creates PRs via Github actions.
		When("create-pr is enabled", func() {
			It("creates a pull request to the remote branch", func() {
				if os.Getenv("GIT_TOKEN") == "" {
					Skip("SKIP, this test needs GIT_TOKEN to work. You really should be running this test!")
				}
				repoLocation := filepath.Join(temp, "repo")
				// clone
				cmd := exec.Command("git", "clone", pctlTestRepositoryName, repoLocation)
				err := cmd.Run()
				Expect(err).ToNot(HaveOccurred())
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd = exec.Command(binaryPath,
					"install",
					"--create-pr",
					"--pr-branch",
					branch,
					"--out",
					repoLocation,
					"--pr-repo",
					"weaveworks/pctl-test-repo",
					"nginx-catalog/weaveworks-nginx")
				session, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Println("Failed output from install: ", string(session))
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("PR created with number:"))
			})

			It("fails if repo is not defined", func() {
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd := exec.Command(binaryPath,
					"install",
					"--create-pr",
					"--pr-branch",
					branch,
					"nginx-catalog/weaveworks-nginx/v0.1.0")
				cmd.Dir = temp
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("repo must be defined if create-pr is true"))
			})

			It("fails if target location is not a git repository", func() {
				if _, ok := os.LookupEnv("GIT_TOKEN"); !ok {
					// Set up a dummy token, because the SCM client is created before we check the git repo.
					err := os.Setenv("GIT_TOKEN", "dummy")
					Expect(err).ToNot(HaveOccurred())
				}
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd := exec.Command(binaryPath,
					"install",
					"--create-pr",
					"--pr-branch",
					branch,
					"--pr-repo",
					"doesnt/matter",
					"nginx-catalog/weaveworks-nginx/v0.1.0")
				cmd.Dir = temp
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("directory is not a git repository"))
			})
		})
	})

	Context("prepare", func() {
		When("dry-run is provided", func() {
			It("displays the to be applied content", func() {
				cmd := exec.Command(binaryPath, "prepare", "--dry-run")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("kind: List"))
			})
		})
		When("baseurl is provided", func() {
			It("will use that to fetch releases", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
				}))
				cmd := exec.Command(binaryPath, "prepare", "--baseurl="+server.URL)
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("status: 418 I'm a teapot"))
			})
		})
		When("version is provided", func() {
			It("will fetch that specific version", func() {
				// use dry-run here so we don't overwrite the created test cluster resources with old version.
				cmd := exec.Command(binaryPath, "prepare", "--version=v0.0.1", "--dry-run")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("kind: List"))
			})
		})
		When("the provided version is missing", func() {
			It("will put out an understandable error message", func() {
				cmd := exec.Command(binaryPath, "prepare", "--version=vnope")
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring("status: 404 Not Found"))
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
