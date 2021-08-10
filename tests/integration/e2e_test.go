package integration_test

import (
	"context"
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
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("end to end flow", func() {
	var (
		gitRepoName             = "pctl-repo"
		profileInstallationName = "e2e"
	)

	BeforeEach(func() {
		var err error
		branch = "flux_repo_test_" + uuid.NewString()[:6]
		namespace = uuid.New().String()
		temp, err = ioutil.TempDir("", "pctl_test_install_upgrade")
		Expect(err).ToNot(HaveOccurred())
		nsp := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(kClient.Create(context.Background(), &nsp)).To(Succeed())

		configMapName = fmt.Sprintf("%s-values", profileInstallationName)
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

		if skipTestsThatRequireCredentials {
			Skip("Skipping this tests as it requires credentials")
		}
		cloneAndCheckoutBranch(temp, branch)

		cmd := exec.Command("flux", "create", "source", "git", gitRepoName, "--url", pctlTestRepositoryHTTP, "--branch", branch, "--namespace", namespace)
		output, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create source git failed: %s", string(output)))
	})

	AfterEach(func() {
		cmd := exec.Command("git", "--git-dir", filepath.Join(temp, ".git"), "--work-tree", temp, "push", "-d", "origin", branch)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("git delete remote branch failed ", string(output))
		}

		cmd = exec.Command("flux", "delete", "source", "git", gitRepoName, "--namespace", namespace, "-s")
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux delete source git failed: %s", string(output)))

		_ = os.RemoveAll(temp)
		deleteNamespace(namespace)
	})

	It("works end-to-end", func() {
		By("searching for the desired weaveworks-nginx profile", func() {
			Expect(pctl("get", "--catalog", "weaveworks-nginx")).To(ContainElement(
				"nginx-catalog/weaveworks-nginx v0.1.0  This installs nginx.",
			))

			Expect(pctl("get", "-p", "v0.1.0", "nginx-catalog/weaveworks-nginx")).To(ContainElements(
				"Catalog       nginx-catalog",
				"Name          weaveworks-nginx",
				"Version       v0.1.0",
				"Description   This installs nginx.",
				"URL           https://github.com/weaveworks/profiles-examples",
				"Maintainer    weaveworks (https://github.com/weaveworks/profiles)",
				"Prerequisites Kubernetes 1.18+",
			))
		})

		By("bootstraping the repo")
		Expect(pctl("bootstrap", "--git-repository", fmt.Sprintf("%s/%s", namespace, gitRepoName), temp)).To(ContainElement("bootstrap completed"))

		By("installing the desired profile", func() {
			pctlAddOutput := pctl(
				"add",
				"--name", profileInstallationName,
				"--namespace", namespace,
				"--config-map", configMapName,
				"nginx-catalog/weaveworks-nginx/v0.1.0",
			)
			Expect(pctlAddOutput).To(ConsistOf(
				"generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0",
				"installation completed successfully",
			))

			By("creating the artifacts")
			profileDir := filepath.Join(temp, "weaveworks-nginx")
			Expect(filesInDir(profileDir)).To(ContainElements(
				"profile-installation.yaml",
				"artifacts/nginx-deployment/kustomization.yaml",
				"artifacts/nginx-deployment/kustomize-flux.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
				"artifacts/nginx-chart/kustomization.yaml",
				"artifacts/nginx-chart/kustomize-flux.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/HelmRelease.yaml",
				"artifacts/nested-profile/nginx-server/kustomization.yaml",
				"artifacts/nested-profile/nginx-server/kustomize-flux.yaml",
			))

			Expect(catFile(filepath.Join(profileDir, "profile-installation.yaml"))).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: %s
  namespace: %s
spec:
  catalog:
    catalog: nginx-catalog
    profile: weaveworks-nginx
    version: v0.1.0
  configMap: %s
  gitRepository:
    name: %s
    namespace: %s
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.0
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, profileInstallationName, namespace, configMapName, gitRepoName, namespace)))

			By("manual editing the profile")
			deploymentFile := filepath.Join(profileDir, "artifacts/nginx-deployment/nginx/deployment/deployment.yaml")
			input := catFile(deploymentFile)
			//the latest version changes this to 3, checking its curerntly 2 for sanity
			Expect(strings.Contains(input, "replicas: 2")).To(BeTrue())
			//sanity check its currently set to 90
			Expect(strings.Contains(input, "containerPort: 80")).To(BeTrue())
			//modify the containerPort to 30
			input = strings.Replace(input, "containerPort: 80", "containerPort: 30", -1)
			Expect(ioutil.WriteFile(deploymentFile, []byte(input), 0644)).To(Succeed())

			By("pushing the profile to the flux repo")
			gitAddAndPush(temp, branch)
			cmd := exec.Command("flux", "reconcile", "source", "git", "--namespace", namespace, gitRepoName)
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux reconcile source git failed : %s", string(output)))

			cmd = exec.Command("flux", "create", "kustomization", "kustomization", "--path", "weaveworks-nginx/", "--source", fmt.Sprintf("GitRepository/%s", gitRepoName), "--namespace", namespace)
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux create kustomization failed : %s", string(output)))

			By("the profile deploying successfully")
			//replicas=3 in v0.1.0
			ensureArtifactDeployedSuccessfully(profileInstallationName, 2)
			By("the profile being returned in pctl get")
			getCmd := func() []string {
				cmd := exec.Command(binaryPath, "get", "--installed", profileInstallationName)
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				return sanitiseString(string(session))
			}

			Eventually(getCmd).Should(ContainElements(
				"INSTALLED PACKAGES",
				"NAMESPACE                            NAME SOURCE                                AVAILABLE UPDATES",
				fmt.Sprintf("%s %s  nginx-catalog/weaveworks-nginx/v0.1.0 v0.1.1", namespace, profileInstallationName),
			))
		})

		By("updating the profile to the latest version", func() {
			Expect(pctl("get", "-p", "v0.1.1", "nginx-catalog/weaveworks-nginx")).To(ContainElements(
				"Catalog       nginx-catalog",
				"Name          weaveworks-nginx",
				"Version       v0.1.1",
				"Description   This installs nginx.",
				"URL           https://github.com/weaveworks/profiles-examples",
				"Maintainer    weaveworks (https://github.com/weaveworks/profiles)",
				"Prerequisites Kubernetes 1.18+",
			))

			profileDir := filepath.Join(temp, "weaveworks-nginx")
			cmd := exec.Command(
				binaryPath,
				"upgrade",
				profileDir,
				"v0.1.1")
			cmd.Dir = temp
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl add failed: %s", string(output)))
			Expect(string(output)).To(ContainSubstring(fmt.Sprintf(`upgrading profile "%s" from version "v0.1.0" to "v0.1.1"`, profileInstallationName)))
			Expect(string(output)).To(ContainSubstring("upgrade completed successfully"))

			var files []string
			err = filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, profileDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ContainElements(
				"profile-installation.yaml",
				"artifacts/nginx-deployment/kustomization.yaml",
				"artifacts/nginx-deployment/kustomize-flux.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
				"artifacts/nginx-chart/kustomization.yaml",
				"artifacts/nginx-chart/kustomize-flux.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/HelmRelease.yaml",
				"artifacts/nested-profile/nginx-server/kustomization.yaml",
				"artifacts/nested-profile/nginx-server/kustomize-flux.yaml",
			))

			filename := filepath.Join(profileDir, "profile-installation.yaml")
			content, err := ioutil.ReadFile(filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: %s
  namespace: %s
spec:
  catalog:
    catalog: nginx-catalog
    profile: weaveworks-nginx
    version: v0.1.1
  configMap: %s
  gitRepository:
    name: %s
    namespace: %s
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.1
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, profileInstallationName, namespace, configMapName, gitRepoName, namespace)))

			By("user changes and upstream changes being applied")
			deploymentFile := filepath.Join(profileDir, "artifacts/nginx-deployment/nginx/deployment/deployment.yaml")
			input, err := ioutil.ReadFile(deploymentFile)
			Expect(err).NotTo(HaveOccurred())

			//the latest version changes this to 3
			Expect(strings.Contains(string(input), "replicas: 3")).To(BeTrue())
			//manual changes should be preserved
			Expect(strings.Contains(string(input), "containerPort: 30")).To(BeTrue())

			By("pushing the updates to the flux repository")
			gitAddAndPush(temp, branch)

			cmd = exec.Command("flux", "reconcile", "source", "git", "--namespace", namespace, gitRepoName)
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("flux reconcile source git failed : %s", string(output)))

			By("the updating succeeding")
			getCmd := func() []string {
				cmd := exec.Command(binaryPath, "get", "--installed", profileInstallationName)
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				return sanitiseString(string(session))
			}
			Eventually(getCmd).Should(ContainElements(
				"INSTALLED PACKAGES",
				"NAMESPACE                            NAME SOURCE                                AVAILABLE UPDATES",
				fmt.Sprintf("%s %s  nginx-catalog/weaveworks-nginx/v0.1.1 -", namespace, profileInstallationName),
			))

			//replicas=3 in v0.1.1
			ensureArtifactDeployedSuccessfully(profileInstallationName, 3)
		})
	})
})

func ensureArtifactDeployedSuccessfully(profileInstallationName string, replicas int) {
	By("successfully deploying the kustomize resource")
	kustomizeName := fmt.Sprintf("%s-%s", profileInstallationName, "nginx-deployment")
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
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		if len(podList.Items) == 0 {
			return v1.PodPhase("no pods found")
		}
		return podList.Items[0].Status.Phase
	}, 2*time.Minute, 5*time.Second).Should(Equal(v1.PodPhase("Running")))
	Expect(podList.Items[0].Spec.Containers[0].Image).To(Equal("nginx:1.14.2"))

	var dep appv1.Deployment
	Eventually(func() int {
		err := kClient.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: "nginx-deployment"}, &dep)
		if err != nil {
			return 0
		}
		return int(*dep.Spec.Replicas)
	}, 2*time.Minute, 5*time.Second).Should(Equal(replicas))

	By("successfully deploying the helmrelease resource")
	helmReleaseName := fmt.Sprintf("%s-%s", profileInstallationName, "nginx-chart")
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
			Name:      fmt.Sprintf("%s-nginx-chart-defaultvalues", profileInstallationName),
			ValuesKey: "default-values.yaml",
		},
		helmv2.ValuesReference{
			Kind:      "ConfigMap",
			Name:      configMapName,
			ValuesKey: "nginx-chart",
		},
	))

	By("successfully deploying the nested helmrelease resource")
	helmReleaseName = fmt.Sprintf("%s-%s", profileInstallationName, "nginx-server")
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
}
