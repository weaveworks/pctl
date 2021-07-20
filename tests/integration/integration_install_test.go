package integration_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	profileExamplesURL                = "https://github.com/weaveworks/profiles-examples"
	pctlPrivateProfilesRepositoryName = "git@github.com:weaveworks/profiles-examples-private.git"
)

var _ = Describe("pctl install", func() {
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
			subName := "pprof"
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
			cmd = exec.Command("flux", "create", "source", "git", gitRepoName, "--url", pctlTestRepositoryHTTP, "--branch", branch, "--namespace", namespace)
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
					"dependon-chart": `master:
  persistence:
    enabled: false
  replica:
    replicaCount: 0`,
				},
			}
			Expect(kClient.Create(context.Background(), &configMap)).To(Succeed())

			cmd = exec.Command(
				binaryPath,
				"install",
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--name",
				subName,
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
				"artifacts/nginx-deployment/kustomization.yaml",
				"artifacts/nginx-deployment/kustomize-flux.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
				"artifacts/nginx-chart/kustomization.yaml",
				"artifacts/nginx-chart/kustomize-flux.yaml",
				"artifacts/dependon-chart/helm-chart/ConfigMap.yaml",
				"artifacts/dependon-chart/helm-chart/HelmRelease.yaml",
				"artifacts/dependon-chart/helm-chart/HelmRepository.yaml",
				"artifacts/dependon-chart/kustomization.yaml",
				"artifacts/dependon-chart/kustomize-flux.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/HelmRelease.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/kustomization.yaml",
				"artifacts/nested-profile/nginx-server/kustomization.yaml",
				"artifacts/nested-profile/nginx-server/kustomize-flux.yaml",
			))

			filename := filepath.Join(temp, "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pprof
  namespace: %s
spec:
  configMap: %s
  source:
    branch: main
    path: weaveworks-nginx
    url: %s
status: {}
`, namespace, configMapName, profileExamplesURL)))

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
					Name:      "pprof-nginx-chart-defaultvalues",
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
			By("successfully deploying the redis resource")
			helmReleaseName = fmt.Sprintf("%s-%s-%s", subName, "weaveworks-nginx", "dependon-chart")
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
			}, 15*time.Minute, 5*time.Second).Should(BeTrue())

			Expect(helmRelease.Spec.ValuesFrom).To(HaveLen(2))
			Expect(helmRelease.Spec.ValuesFrom).To(ConsistOf(
				helmv2.ValuesReference{
					Kind:      "ConfigMap",
					Name:      "pprof-dependon-chart-defaultvalues",
					ValuesKey: "default-values.yaml",
				},
				helmv2.ValuesReference{
					Kind:      "ConfigMap",
					Name:      configMapName,
					ValuesKey: "dependon-chart",
				},
			))
		})

		When("a url is provided with a branch and path", func() {
			It("will fetch information from that branch with path", func() {
				namespace := uuid.New().String()
				branch := "main"
				path := "bitnami-nginx"
				cmd := exec.Command(
					binaryPath,
					"install",
					"--git-repository",
					namespace+"/git-repo-name",
					"--namespace",
					namespace,
					"--profile-url",
					profileExamplesURL,
					"--profile-branch",
					branch,
					"--profile-path",
					path,
				)
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
				Expect(files).To(ContainElements(
					temp,
					filepath.Join(temp, "profile-installation.yaml"),
					filepath.Join(temp, "artifacts"),
					filepath.Join(temp, "artifacts", "nginx-server"),
					filepath.Join(temp, "artifacts", "nginx-server", "helm-chart", "HelmRelease.yaml"),
					filepath.Join(temp, "artifacts", "nginx-server", "helm-chart", "kustomization.yaml"),
					filepath.Join(temp, "artifacts", "nginx-server", "helm-chart", "nginx", "chart", "Chart.yaml"),
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
    url: %s
status: {}
`, namespace, profileExamplesURL)))
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
				Expect(files).To(ContainElements(
					temp,
					filepath.Join(temp, "profile-installation.yaml"),
					filepath.Join(temp, "artifacts", "nginx-server", "helm-chart", "nginx", "chart", "Chart.yaml"),
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
				cmd := exec.Command(
					binaryPath,
					"install",
					"--git-repository",
					namespace+"/git-repo-name",
					"--namespace",
					namespace,
					"--profile-url",
					profileExamplesURL,
					"--profile-branch",
					branch,
					"--profile-path",
					path,
					"catalog/profile/v0.0.1",
				)
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
					pctlTestRepositoryOrgName,
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
			By("creating the flux repository")
			gitRepoName := "pctl-repo"
			branch := "flux_repo_test_" + uuid.NewString()[:6]

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
			cmd = exec.Command("flux", "create", "source", "git", gitRepoName, "--url", pctlTestRepositoryHTTP, "--branch", branch, "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create source git failed: %s", string(output)))

			By("running the install")
			cmd = exec.Command(binaryPath, "install", "--namespace", namespace, "--git-repository", fmt.Sprintf("%s/%s", namespace, gitRepoName), "nginx-catalog/nginx/v2.0.1")
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
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
				"artifacts/bitnami-nginx/helm-chart/ConfigMap.yaml",
				"artifacts/bitnami-nginx/helm-chart/HelmRelease.yaml",
				"artifacts/bitnami-nginx/helm-chart/HelmRepository.yaml",
				"artifacts/bitnami-nginx/kustomization.yaml",
				"artifacts/bitnami-nginx/kustomize-flux.yaml",
				"profile-installation.yaml",
			))

			filename := filepath.Join(temp, "nginx", "profile-installation.yaml")
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
		When("there is a depending chart", func() {
			It("generates artifacts which contain a depends on flag", func() {
				cmd := exec.Command(
					binaryPath,
					"install",
					"--git-repository",
					namespace+"/git-repo-name",
					"--namespace",
					namespace,
					"--profile-url",
					profileExamplesURL,
					"--profile-path",
					"dependson-nginx",
				)
				cmd.Dir = temp
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed : %s", string(output)))
				Expect(string(output)).To(ContainSubstring(fmt.Sprintf("generating profile installation from source: repository %s, path: dependson-nginx and branch main", profileExamplesURL)))

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
					"artifacts/dependon/nginx/deployment/deployment.yaml",
					"artifacts/dependon/kustomization.yaml",
					"artifacts/dependon/kustomize-flux.yaml",
					"artifacts/dependon2/helm-chart/ConfigMap.yaml",
					"artifacts/dependon2/helm-chart/HelmRelease.yaml",
					"artifacts/dependon2/helm-chart/HelmRepository.yaml",
					"artifacts/dependon2/kustomization.yaml",
					"artifacts/dependon2/kustomize-flux.yaml",
					"artifacts/nginx-chart/helm-chart/ConfigMap.yaml",
					"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
					"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
					"artifacts/nginx-chart/kustomization.yaml",
					"artifacts/nginx-chart/kustomize-flux.yaml",
					"profile-installation.yaml",
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
    path: dependson-nginx
    url: %s
status: {}
`, namespace, profileExamplesURL)))

				By("verify that dependsOn has been added to the kustomize resource")
				filename = filepath.Join(temp, "artifacts", "nginx-chart", "kustomize-flux.yaml")
				content, err = ioutil.ReadFile(filename)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  creationTimestamp: null
  name: pctl-profile-dependson-nginx-nginx-chart
  namespace: %s
spec:
  dependsOn:
  - name: pctl-profile-dependson-nginx-dependon
    namespace: %s
  - name: pctl-profile-dependson-nginx-dependon2
    namespace: %s
  healthChecks:
  - apiVersion: helm.toolkit.fluxcd.io/v2beta1
    kind: HelmRelease
    name: pctl-profile-dependson-nginx-nginx-chart
    namespace: %s
  interval: 5m0s
  path: artifacts/nginx-chart/helm-chart
  prune: true
  sourceRef:
    kind: GitRepository
    name: git-repo-name
    namespace: %s
  targetNamespace: %s
status: {}
`, namespace, namespace, namespace, namespace, namespace, namespace)))
				// This is a bit more involved since kubectl apply -r no longer works. Depends on only works with Flux.
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
