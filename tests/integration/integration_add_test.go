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

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/dependency"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	profileExamplesURL                = "https://github.com/weaveworks/profiles-examples"
	kivoPrivateProfilesRepositoryName = "git@github.com:weaveworks/profiles-examples-private.git"
)

var _ = Describe("kivo add", func() {
	BeforeEach(func() {
		namespace = uuid.New().String()
		createNamespace(namespace)

		branch = "flux_repo_test_" + uuid.NewString()[:6]
	})

	AfterEach(func() {
		cmd := exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-d", "origin", branch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("git delete remote branch failed ", string(output))
		}

		deleteNamespace(namespace)
	})

	It("generates valid artifacts with the correct dependency ordering", func() {
		if skipTestsThatRequireCredentials {
			Skip("Skipping this tests as it requires credentials")
		}

		profileBranch := "main"
		subName := "pprof"
		gitRepoName := "kivo-repo"

		cloneAndCheckoutBranch(temp, branch)

		// setup the gitrepository resources. Requires the branch to exist first
		cmd := exec.Command("flux", "create", "source", "git", gitRepoName, "--url", kivoTestRepositoryHTTP, "--branch", branch, "--namespace", namespace)
		output, err := cmd.CombinedOutput()
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

		args := []string{
			"add",
			"--name", subName,
			"--git-repository",
			fmt.Sprintf("%s/%s", namespace, gitRepoName),
			"--namespace", namespace,
			"--name",
			subName,
			"--profile-branch",
			profileBranch,
			"--profile-repo-url", profileExamplesURL,
			"--profile-path", "weaveworks-nginx",
			"--config-map", configMapName,
		}
		Expect(kivo(args...)).To(ContainElement(
			fmt.Sprintf("► generating profile installation from source: repository %s, path: %s and branch %s", profileExamplesURL, "weaveworks-nginx", profileBranch),
		))

		profilesDir := filepath.Join(temp, subName)
		By("creating the artifacts")
		Expect(filesInDir(profilesDir)).To(ContainElements(
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

		Expect(catFile(filepath.Join(profilesDir, "profile-installation.yaml"))).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: pprof
  namespace: %s
spec:
  configMap: %s
  gitRepository:
    name: %s
    namespace: %s
  source:
    branch: %s
    path: weaveworks-nginx
    url: %s
status: {}
`, namespace, configMapName, gitRepoName, namespace, profileBranch, profileExamplesURL)))

		By("the artifacts being deployable")
		// Generate the resources into the flux repo, and push them up the repo?
		gitAddAndPush(temp, branch)

		cmd = exec.Command("flux", "reconcile", "source", "git", gitRepoName, "--namespace", namespace)
		cmd.Dir = temp
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux reconcile source git failed : %s", string(output)))

		cmd = exec.Command("flux", "create", "kustomization", "kustomization", "--source", fmt.Sprintf("GitRepository/%s", gitRepoName), "--path", ".", "--namespace", namespace)
		cmd.Dir = temp
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create kustomization failed : %s", string(output)))

		By("successfully deploying the kustomize resource")
		kustomizeName := fmt.Sprintf("%s-%s", subName, "nginx-deployment")
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
		helmReleaseName := fmt.Sprintf("%s-%s", subName, "nginx-chart")
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
		helmReleaseName = fmt.Sprintf("%s-%s", subName, "nginx-server")
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

		By("successfully deploying the redis resource with dependsOn")
		kustomizeName = fmt.Sprintf("%s-%s", subName, "dependon-chart")
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
		}, 15*time.Minute, 5*time.Second).Should(BeTrue())

		Expect(kustomize.Spec.DependsOn).To(ConsistOf(
			dependency.CrossNamespaceDependencyReference{
				Name:      fmt.Sprintf("%s-%s", subName, "nginx-deployment"),
				Namespace: namespace,
			},
			dependency.CrossNamespaceDependencyReference{
				Name:      fmt.Sprintf("%s-%s", subName, "nginx-chart"),
				Namespace: namespace,
			},
			dependency.CrossNamespaceDependencyReference{
				Name:      fmt.Sprintf("%s-%s", subName, "nginx-server"),
				Namespace: namespace,
			},
		))

		helmReleaseName = fmt.Sprintf("%s-%s", subName, "dependon-chart")
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

		cmd = exec.Command("flux", "delete", "source", "git", gitRepoName, "--namespace", namespace, "-s")
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux delete source git failed: %s", string(output)))
	})

	When("a url is provided with a branch and path", func() {
		It("will fetch information from that branch with path", func() {
			namespace := uuid.New().String()
			branch := "main"
			path := "bitnami-nginx"
			args := []string{
				"add",
				"--name", "kivo-profile",
				"--git-repository",
				namespace + "/git-repo-name",
				"--namespace",
				namespace,
				"--profile-repo-url",
				profileExamplesURL,
				"--profile-branch",
				branch,
				"--profile-path",
				path,
			}
			Expect(kivo(args...)).To(ContainElement("✔ installation completed successfully"))
			By("creating the artifacts")
			Expect(filesInDir(temp)).To(ContainElements(
				"kivo-profile/profile-installation.yaml",
				filepath.Join("kivo-profile", "artifacts", "nginx-server", "helm-chart", "HelmRelease.yaml"),
				filepath.Join("kivo-profile", "artifacts", "nginx-server", "helm-chart", "kustomization.yaml"),
				filepath.Join("kivo-profile", "artifacts", "nginx-server", "helm-chart", "nginx", "chart", "Chart.yaml"),
			))
			filename := filepath.Join(temp, "kivo-profile", "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: kivo-profile
  namespace: %s
spec:
  gitRepository:
    name: git-repo-name
    namespace: %s
  source:
    branch: main
    path: bitnami-nginx
    url: %s
status: {}
`, namespace, namespace, profileExamplesURL)))
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
			cmd := exec.Command(binaryPath, "add", "--name", "kivo-profile", "--out", temp, "--git-repository", namespace+"/git-repo-name", "--namespace", namespace, "--profile-repo-url", kivoPrivateProfilesRepositoryName, "--profile-branch", branch, "--profile-path", path)
			cmd.Dir = temp

			if v := os.Getenv("PRIVATE_EXAMPLES_DEPLOY_KEY"); v != "" {
				cmd.Env = append(cmd.Env, os.Environ()...)
				cmd.Env = append(cmd.Env, fmt.Sprintf(`GIT_SSH_COMMAND="ssh -i %s"`, v))
			}
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("kivo add failed : %s", string(output)))

			By("creating the artifacts")
			Expect(filesInDir(temp)).To(ContainElements(
				"kivo-profile/profile-installation.yaml",
				filepath.Join("kivo-profile", "artifacts", "nginx-server", "helm-chart", "nginx", "chart", "Chart.yaml"),
			))
			filename := filepath.Join(temp, "kivo-profile", "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: kivo-profile
  namespace: %s
spec:
  gitRepository:
    name: git-repo-name
    namespace: %s
  source:
    branch: main
    path: bitnami-nginx
    url: git@github.com:weaveworks/profiles-examples-private.git
status: {}
`, namespace, namespace)))
		})
	})

	When("url and catalog entry add format are both defined", func() {
		It("will throw a meaningful error", func() {
			namespace := uuid.New().String()
			branch := "branch-and-url"
			path := "branch-nginx"
			cmd := exec.Command(
				binaryPath,
				"add",
				"--name", "kivo-profile",
				"--git-repository",
				namespace+"/git-repo-name",
				"--namespace",
				namespace,
				"--profile-repo-url",
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

	When("git repository is not provided", func() {
		It("will throw a meaningful error", func() {
			namespace := uuid.New().String()
			branch := "branch-and-url"
			path := "branch-nginx"
			cmd := exec.Command(
				binaryPath,
				"add",
				"--name", "kivo-profile",
				"--namespace",
				namespace,
				"--profile-repo-url",
				profileExamplesURL,
				"--profile-branch",
				branch,
				"--profile-path",
				path,
			)
			cmd.Dir = temp
			session, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(session)).To(ContainSubstring("flux git repository not provided, please provide the --git-repository flag or use the kivo bootstrap functionality"))
		})
	})

	When("a catalog version is provided, but it's an invalid/missing version", func() {
		It("provide an error saying the profile with these specifics can't be found", func() {
			cmd := exec.Command(binaryPath, "add", "--name", "kivo-profile", "--git-repository", namespace+"/git-repo-name", "nginx-catalog/weaveworks-nginx/v999.9.9")
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring(`unable to find profile "weaveworks-nginx" in catalog "nginx-catalog" (with version if provided: v999.9.9)`))
		})
	})

	When("missing arguments when url is not provided", func() {
		It("returns an error message", func() {
			cmd := exec.Command(binaryPath, "add", "--name", "pctl-profile", "--git-repository", namespace+"/git-repo-name")
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(string(output)).To(ContainSubstring(`<CATALOG>/<PROFILE>[/<VERSION>] must be provided`))
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
			cmd := exec.Command("git", "clone", kivoTestRepositoryName, repoLocation)
			err := cmd.Run()
			Expect(err).ToNot(HaveOccurred())
			suffix, err := randString(3)
			Expect(err).NotTo(HaveOccurred())
			branch := "prtest_" + suffix
			cmd = exec.Command(binaryPath,
				"add",
				"--name", "kivo-profile",
				"--git-repository", namespace+"/git-repo-name",
				"--create-pr",
				"--pr-branch",
				branch,
				"--out",
				repoLocation,
				"--pr-repo",
				kivoTestRepositoryOrgName,
				"nginx-catalog/weaveworks-nginx")
			cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", os.Getenv("GIT_TOKEN")))
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("kivo add failed : %s", string(output)))
			Expect(string(output)).To(ContainSubstring("PR created with number:"))
		})

		It("fails if repo is not defined", func() {
			suffix, err := randString(3)
			Expect(err).NotTo(HaveOccurred())
			branch := "prtest_" + suffix
			cmd := exec.Command(
				binaryPath,
				"add",
				"--name", "kivo-profile",
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
			Expect(os.Setenv("GITHUB_TOKEN", "dummy")).To(Succeed())
			suffix, err := randString(3)
			Expect(err).NotTo(HaveOccurred())
			branch := "prtest_" + suffix
			cmd := exec.Command(
				binaryPath,
				"add",
				"--name", "kivo-profile",
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

	Context("adding from a single-profile repo", func() {
		var (
			configMapName string
			subName       string
		)

		BeforeEach(func() {
			subName = "kivo-profile"
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
		})

		It("generates valid artifacts to the local directory", func() {
			if skipTestsThatRequireCredentials {
				Skip("Skipping this tests as it requires credentials")
			}
			By("creating the flux repository")
			gitRepoName := "kivo-repo"
			branch := "flux_repo_test_" + uuid.NewString()[:6]

			cloneAndCheckoutBranch(temp, branch)
			// setup the gitrepository resources. Requires the branch to exist first
			cmd := exec.Command("flux", "create", "source", "git", gitRepoName, "--url", kivoTestRepositoryHTTP, "--branch", branch, "--namespace", namespace)
			cmd.Dir = temp
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create source git failed: %s", string(output)))

			By("adding the profile")
			args := []string{
				"add",
				"--name", subName,
				"--namespace", namespace,
				"--git-repository", fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"nginx-catalog/nginx/v2.0.1",
			}
			Expect(kivo(args...)).To(ContainElement("► generating profile installation from source: catalog entry nginx-catalog/nginx/v2.0.1"))

			By("creating the artifacts")
			profilesDir := filepath.Join(temp, subName)
			Expect(filesInDir(profilesDir)).To(ContainElements(
				"artifacts/bitnami-nginx/helm-chart/ConfigMap.yaml",
				"artifacts/bitnami-nginx/helm-chart/HelmRelease.yaml",
				"artifacts/bitnami-nginx/helm-chart/HelmRepository.yaml",
				"artifacts/bitnami-nginx/kustomization.yaml",
				"artifacts/bitnami-nginx/kustomize-flux.yaml",
				"profile-installation.yaml",
			))

			Expect(catFile(filepath.Join(profilesDir, "profile-installation.yaml"))).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: kivo-profile
  namespace: %s
spec:
  catalog:
    catalog: nginx-catalog
    profile: nginx
    version: v2.0.1
  gitRepository:
    name: %s
    namespace: %s
  source:
    path: .
    tag: v2.0.1
    url: https://github.com/weaveworks/nginx-profile
status: {}
`, namespace, gitRepoName, namespace)))

			By("the artifacts being deployable")
			gitAddAndPush(temp, branch)

			cmd = exec.Command("flux", "reconcile", "source", "git", gitRepoName, "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux reconcile source git failed : %s", string(output)))

			cmd = exec.Command("flux", "create", "kustomization", "kustomization", "--source", fmt.Sprintf("GitRepository/%s", gitRepoName), "--path", ".", "--namespace", namespace)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create kustomization failed : %s", string(output)))

			By("successfully deploying the kustomize resource")
			helmReleaseName := fmt.Sprintf("%s-%s", subName, "bitnami-nginx")
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
			}, 10*time.Minute, 5*time.Second).Should(Equal(v1.PodPhase("Running")))

			Expect(podList.Items[0].Spec.Containers[0].Image).To(Equal("docker.io/bitnami/nginx:1.19.10-debian-10-r35"))
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
