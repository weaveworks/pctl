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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	profileExamplesURL                = "https://github.com/weaveworks/profiles-examples"
	pctlPrivateProfilesRepositoryName = "git@github.com:weaveworks/profiles-examples-private.git"
)

var _ = Describe("PCTL", func() {
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

	Context("list", func() {
		var (
			namespace        = "default"
			installationName = "long-name-to-ensure-padding"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileInstallation
		)

		BeforeEach(func() {
			profileURL := "https://github.com/weaveworks/profiles-examples"
			pSub = profilesv1.ProfileInstallation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileInstallation",
					APIVersion: "profileinstallations.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      installationName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL: profileURL,
						Tag: "weaveworks-nginx/v0.1.0",
					},
					Catalog: &profilesv1.Catalog{
						Catalog: "nginx-catalog",
						Profile: "weaveworks-nginx",
						Version: "v0.1.0",
					},
				},
			}
			Expect(kClient.Create(ctx, &pSub)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(kClient.Delete(ctx, &pSub)).Should(Succeed())
		})

		It("returns the installations", func() {
			listCmd := func() []string {
				cmd := exec.Command(binaryPath, "list")
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				return strings.Split(string(session), "\n")
			}

			Eventually(listCmd).Should(ContainElements(
				"NAMESPACE\tNAME                       \tSOURCE                               \tAVAILABLE UPDATES ",
				"default  \tlong-name-to-ensure-padding\tnginx-catalog/weaveworks-nginx/v0.1.0\tv0.1.1           \t",
			))
		})

		When("there are no available updates", func() {
			It("returns the installations", func() {
				profileURL := "https://github.com/weaveworks/profiles-examples"
				bitnamiSub := profilesv1.ProfileInstallation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileInstallation",
						APIVersion: "profileinstallations.weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bitnami-profile",
						Namespace: namespace,
					},
					Spec: profilesv1.ProfileInstallationSpec{
						Source: &profilesv1.Source{
							URL: profileURL,
							Tag: "bitnami-nginx/v0.0.1",
						},
						Catalog: &profilesv1.Catalog{
							Catalog: "nginx-catalog",
							Profile: "bitnami-nginx",
							Version: "v0.0.1",
						},
					},
				}
				Expect(kClient.Create(ctx, &bitnamiSub)).Should(Succeed())
				listCmd := func() []string {
					cmd := exec.Command(binaryPath, "list")
					session, err := cmd.CombinedOutput()
					Expect(err).ToNot(HaveOccurred())
					return strings.Split(string(session), "\n")
				}

				Eventually(listCmd).Should(ContainElements(
					"NAMESPACE\tNAME                       \tSOURCE                               \tAVAILABLE UPDATES ",
					"default  \tbitnami-profile            \tnginx-catalog/bitnami-nginx/v0.0.1   \t-                \t",
					"default  \tlong-name-to-ensure-padding\tnginx-catalog/weaveworks-nginx/v0.1.0\tv0.1.1           \t",
				))
				Expect(kClient.Delete(ctx, &bitnamiSub)).Should(Succeed())
			})
		})
	})

	Context("install", func() {
		var (
			temp      string
			namespace string
			branch    string
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

			branch = "flux_repo_test_" + uuid.NewString()[:6]
		})

		AfterEach(func() {
			cmd := exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-d", "origin", branch)
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println("git delete remote branch failed ", string(output))
			}

			_ = os.RemoveAll(temp)
			nsp := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_ = kClient.Delete(context.Background(), &nsp)

		})

		It("generates valid artifacts to the local directory", func() {
			if skipTestsThatRequireCredentials {
				Skip("Skipping this tests as it requires credentials")
			}

			profileBranch := "main"
			subName := "pctl-profile"
			gitRepoName := "pctl-repo"

			// check out the branch
			cmd := exec.Command("git", "clone", pctlTestRepositoryName, temp)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("clone failed: %s", string(output)))
			cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "checkout", "-b", branch)
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("checkout branch failed: %s", string(output)))

			cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-u", "origin", branch)
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("git push failed : %s", string(output)))
			// setup the gitrepository resources. Requires the branch to exist first
			cmd = exec.Command("flux", "create", "source", "git", gitRepoName, "--url", "https://github.com/weaveworks/pctl-test-repo", "--branch", branch, "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create source git failed: %s", string(output)))

			configMapName := subName + "-values"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"nginx-server": `replicas: 2`,
					"nginx-chart":  `replicas: 2`,
				},
			}
			Expect(kClient.Create(context.Background(), &configMap)).To(Succeed())

			cmd = exec.Command(
				binaryPath,
				"install",
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--profile-branch",
				profileBranch,
				"--profile-url", profileExamplesURL,
				"--profile-path", "weaveworks-nginx",
				"--config-map", configMapName)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed: %s", string(output)))
			Expect(string(output)).To(ContainSubstring(
				fmt.Sprintf("generating profile installation from source: repository %s, path: %s and branch %s", profileExamplesURL, "weaveworks-nginx", profileBranch),
			))

			var files []string
			profilesDir := filepath.Join(temp)
			err = filepath.Walk(profilesDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, profilesDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating the artifacts")
			Expect(files).To(ContainElements(
				"profile-installation.yaml",
				"artifacts/nginx-deployment/Kustomization.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/HelmRepository.yaml",
				"artifacts/nested-profile/nginx-server/HelmRelease.yaml",
			))

			filename := filepath.Join(temp, "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  configMap: %s
  source:
    branch: main
    path: weaveworks-nginx
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName)))

			By("the artifacts being deployable")
			// Generate the resources into the flux repo, and push them up the repo?
			cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "add", ".")
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("git add . failed: %s", string(output)))
			cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "commit", "-am", "new content")
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("git commit failed : %s", string(output)))
			cmd = exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-u", "origin", branch)
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("git push failed : %s", string(output)))

			cmd = exec.Command("flux", "reconcile", "source", "git", gitRepoName, "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux reconcile source git failed : %s", string(output)))

			cmd = exec.Command("flux", "create", "kustomization", "kustomization", "--source", fmt.Sprintf("GitRepository/%s", gitRepoName), "--path", ".", "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create kustomization failed : %s", string(output)))

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
			var podList *v1.PodList
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

			By("successfully deploying the helmrelease resource")
			helmReleaseName := fmt.Sprintf("%s-%s-%s", subName, "weaveworks-nginx", "nginx-chart")
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
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			Expect(helmRelease.Spec.ValuesFrom).To(HaveLen(2))
			Expect(helmRelease.Spec.ValuesFrom).To(ConsistOf(
				helmv2.ValuesReference{
					Kind:      "ConfigMap",
					Name:      "pctl-profile-nginx-chart-defaultvalues",
					ValuesKey: "default-values.yaml",
				},
				helmv2.ValuesReference{
					Kind:      "ConfigMap",
					Name:      configMapName,
					ValuesKey: "nginx-chart",
				},
			))

			By("successfully deploying the nested helmrelease resource")
			helmReleaseName = fmt.Sprintf("%s-%s-%s", subName, "bitnami-nginx", "nginx-server")
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
			}, 5*time.Minute, 5*time.Second).Should(BeTrue())

			Expect(helmRelease.Spec.ValuesFrom).To(HaveLen(1))
			Expect(helmRelease.Spec.ValuesFrom[0]).To(Equal(helmv2.ValuesReference{
				Kind:      "ConfigMap",
				Name:      configMapName,
				ValuesKey: "nginx-server",
			}))
		})

		When("a url is provided with a branch and path", func() {
			It("will fetch information from that branch with path", func() {
				namespace := uuid.New().String()
				branch := "main"
				path := "bitnami-nginx"
				cmd := exec.Command(binaryPath, "install", "--git-repository", namespace+"/git-repo-name", "--namespace", namespace, "--profile-url", "https://github.com/weaveworks/profiles-examples", "--profile-branch", branch, "--profile-path", path)
				cmd.Dir = temp
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed : %s", string(output)))

				var files []string
				err = filepath.Walk(temp, func(path string, info os.FileInfo, err error) error {
					files = append(files, path)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				By("creating the artifacts")
				profilesDirProfile := filepath.Join(temp, "profile-installation.yaml")
				profilesArtifacts := filepath.Join(temp, "artifacts")
				profilesArtifactsChartDir := filepath.Join(profilesArtifacts, "nginx-server")
				profilesArtifactsRelease := filepath.Join(profilesArtifactsChartDir, "HelmRelease.yaml")
				profilesArtifactsChart := filepath.Join(profilesArtifactsChartDir, "nginx", "chart", "Chart.yaml")
				Expect(files).To(ContainElements(
					temp,
					profilesDirProfile,
					profilesArtifacts,
					profilesArtifactsChartDir,
					profilesArtifactsChart,
					profilesArtifactsRelease,
				))
				filename := filepath.Join(temp, "profile-installation.yaml")
				content, err := ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  source:
    branch: main
    path: bitnami-nginx
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace)))
			})
		})

		When("a url is provided to a private repository", func() {
			It("will fetch information without a problem", func() {
				if skipTestsThatRequireCredentials {
					Skip("Skipping this tests as it requires credentials")
				}
				namespace := uuid.New().String()
				branch := "main"
				path := "bitnami-nginx"
				cmd := exec.Command(binaryPath, "install", "--out", temp, "--git-repository", namespace+"/git-repo-name", "--namespace", namespace, "--profile-url", pctlPrivateProfilesRepositoryName, "--profile-branch", branch, "--profile-path", path)
				cmd.Dir = temp

				if v := os.Getenv("PRIVATE_EXAMPLES_DEPLOY_KEY"); v != "" {
					cmd.Env = append(cmd.Env, os.Environ()...)
					cmd.Env = append(cmd.Env, fmt.Sprintf(`GIT_SSH_COMMAND="ssh -i %s"`, v))
				}
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed : %s", string(output)))

				var files []string
				err = filepath.Walk(temp, func(path string, info os.FileInfo, err error) error {
					files = append(files, path)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				By("creating the artifacts")
				profilesDirProfile := filepath.Join(temp, "profile-installation.yaml")
				profilesArtifacts := filepath.Join(temp, "artifacts")
				profilesArtifactsDeployment := filepath.Join(temp, "artifacts", "nginx-server")
				profilesArtifactsDeploymentKustomizationNginx := filepath.Join(temp, "artifacts", "nginx-server", "nginx")
				profilesArtifactsDeploymentKustomizationNginxChart := filepath.Join(temp, "artifacts", "nginx-server", "nginx", "chart")
				profilesArtifactsDeploymentKustomizationNginxChartYaml := filepath.Join(temp, "artifacts", "nginx-server", "nginx", "chart", "Chart.yaml")
				Expect(files).To(ContainElements(
					temp,
					profilesDirProfile,
					profilesArtifacts,
					profilesArtifactsDeployment,
					profilesArtifactsDeploymentKustomizationNginx,
					profilesArtifactsDeploymentKustomizationNginxChart,
					profilesArtifactsDeploymentKustomizationNginxChartYaml,
				))
				filename := filepath.Join(temp, "profile-installation.yaml")
				content, err := ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  source:
    branch: main
    path: bitnami-nginx
    url: git@github.com:weaveworks/profiles-examples-private.git
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
				cmd := exec.Command(binaryPath, "install", "--git-repository", namespace+"/git-repo-name", "--namespace", namespace, "--profile-url", "https://github.com/weaveworks/profiles-examples", "--profile-branch", branch, "--profile-path", path, "catalog/profile/v0.0.1")
				cmd.Dir = temp
				session, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(session)).To(ContainSubstring("it looks like you provided a url with a catalog entry; please choose either format: url/branch/path or <CATALOG>/<PROFILE>[/<VERSION>]"))
			})
		})

		When("a catalog version is provided, but it's an invalid/missing version", func() {
			It("provide an error saying the profile with these specifics can't be found", func() {
				cmd := exec.Command(binaryPath, "install", "--git-repository", namespace+"/git-repo-name", "nginx-catalog/weaveworks-nginx/v999.9.9")
				output, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(output)).To(ContainSubstring(`unable to find profile "weaveworks-nginx" in catalog "nginx-catalog" (with version if provided: v999.9.9)`))
			})
		})

		// Note, the repo cleans the creates PRs via Github actions.
		When("create-pr is enabled", func() {
			It("creates a pull request to the remote branch", func() {
				if skipTestsThatRequireCredentials {
					Skip("Skipping this tests as it requires credentials")
				}
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
					"--git-repository", namespace+"/git-repo-name",
					"--create-pr",
					"--pr-branch",
					branch,
					"--out",
					repoLocation,
					"--pr-repo",
					"weaveworks/pctl-test-repo",
					"nginx-catalog/weaveworks-nginx")
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed : %s", string(output)))
				Expect(string(output)).To(ContainSubstring("PR created with number:"))
			})

			It("fails if repo is not defined", func() {
				suffix, err := randString(3)
				Expect(err).NotTo(HaveOccurred())
				branch := "prtest_" + suffix
				cmd := exec.Command(binaryPath,
					"install",
					"--git-repository", namespace+"/git-repo-name",
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
				if skipTestsThatRequireCredentials {
					Skip("Skipping this tests as it requires credentials")
				}
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
					"--git-repository", namespace+"/git-repo-name",
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

	Context("installing from a single-profile repo", func() {
		var (
			temp          string
			namespace     string
			configMapName string
			subName       string
		)

		BeforeEach(func() {
			var err error
			subName = "pctl-profile"
			namespace = uuid.New().String()
			temp, err = ioutil.TempDir("", "pctl_test_install_single_profile_01")
			Expect(err).ToNot(HaveOccurred())
			nsp := v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(kClient.Create(context.Background(), &nsp)).To(Succeed())
			configMapName = subName + "-values"
			configMap := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: namespace,
				},
				Data: map[string]string{
					"values.yaml": `service:
  type: ClusterIP`,
				},
			}
			Expect(kClient.Create(context.Background(), &configMap)).To(Succeed())
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
			cmd := exec.Command(binaryPath, "install", "--namespace", namespace, "nginx-catalog/nginx/v2.0.1")
			cmd.Dir = temp
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed : %s", string(output)))
			Expect(string(output)).To(ContainSubstring("generating profile installation from source: catalog entry nginx-catalog/nginx/v2.0.1"))

			var files []string
			profilesDir := filepath.Join(temp, "nginx")
			err = filepath.Walk(profilesDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, profilesDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating the artifacts")
			Expect(files).To(ContainElements(
				"nginx/artifacts/bitnami-nginx/ConfigMap.yaml",
				"nginx/artifacts/bitnami-nginx/HelmRelease.yaml",
				"nginx/artifacts/bitnami-nginx/HelmRepository.yaml",
				"nginx/profile-installation.yaml",
			))

			filename := filepath.Join(temp, "nginx", "nginx", "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pctl-profile
  namespace: %s
spec:
  catalog:
    catalog: nginx-catalog
    profile: nginx
    version: v2.0.1
  source:
    path: .
    tag: v2.0.1
    url: https://github.com/weaveworks/nginx-profile
status: {}
`, namespace)))

			By("the artifacts being deployable")

			cmd = exec.Command("kubectl", "apply", "-R", "-f", profilesDir)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("kubectl apply failed : %s", string(output)))

			By("successfully deploying the kustomize resource")
			helmReleaseName := fmt.Sprintf("%s-%s-%s", subName, "nginx", "bitnami-nginx")
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
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			helmReleaseOpts := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{"helm.sh/chart": "nginx-8.9.1"},
			}
			podList := &v1.PodList{}
			Eventually(func() v1.PodPhase {
				podList = &v1.PodList{}
				err := kClient.List(context.Background(), podList, helmReleaseOpts...)
				Expect(err).NotTo(HaveOccurred())
				if len(podList.Items) == 0 {
					return v1.PodPhase("no pods found")
				}
				return podList.Items[0].Status.Phase
			}, 2*time.Minute, 5*time.Second).Should(Equal(v1.PodPhase("Running")))

			Expect(podList.Items[0].Spec.Containers[0].Image).To(Equal("docker.io/bitnami/nginx:1.19.10-debian-10-r35"))
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

	Context("docgen", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "docgen")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = os.RemoveAll(tmpDir)
		})

		It("writes pctl command help to markdown files", func() {
			cmd := exec.Command(binaryPath, "docgen", "--path", tmpDir)
			_, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			files, err := ioutil.ReadDir(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).To(Equal(5))
			commands := []string{"install", "prepare", "list", "show", "search"}
			for _, cmd := range commands {
				filename := filepath.Join(tmpDir, fmt.Sprintf("pctl-%s-cmd.md", cmd))
				Expect(filename).To(BeAnExistingFile())
				contents, err := ioutil.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(cmd))
				Expect(string(contents)).To(ContainSubstring("NAME"))
				Expect(string(contents)).To(ContainSubstring("USAGE"))
			}
		})

		When("the provided output directory does not exist", func() {
			It("creates it", func() {
				newDir := filepath.Join(tmpDir, "does-not-exist")
				Expect(newDir).ToNot(BeAnExistingFile())

				cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
				_, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(newDir).To(BeAnExistingFile())
			})

			When("creating the docs dir fails", func() {
				It("exits 1", func() {
					newDir := filepath.Join(tmpDir, "does-not-exist")
					Expect(os.Chmod(tmpDir, 0600)).To(Succeed())

					cmd := exec.Command(binaryPath, "docgen", "--path", newDir)
					_, err := cmd.CombinedOutput()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("writing the docs files fails", func() {
			It("exits 1", func() {
				Expect(os.Chmod(tmpDir, 0600)).To(Succeed())

				cmd := exec.Command(binaryPath, "docgen", "--path", tmpDir)
				_, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
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
