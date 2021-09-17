package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("upgrade", func() {
	BeforeEach(func() {
		namespace = uuid.New().String()
		nsp := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(kClient.Create(context.Background(), &nsp)).To(Succeed())

		configMapName = "pctl-profile-values"
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

	})

	AfterEach(func() {
		deleteNamespace(namespace)
	})

	When("using the latest flag with available updates", func() {
		It("upgrades to the latest available version", func() {
			gitRepoName := "my-git-repo"
			subname := "pctl-profile"
			profileDir := filepath.Join(temp, subname)
			args := []string{
				"add",
				"--name", subname,
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--config-map", configMapName,
				"nginx-catalog/weaveworks-nginx/v0.1.0",
			}

			Expect(pctl(args...)).To(ContainElements(
				"► generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0",
				"✔ installation completed successfully",
			))
			By("upgrading a profile")
			args = []string{
				"upgrade",
				"--latest",
				profileDir,
			}
			Expect(pctl(args...)).To(ContainElements(
				`► upgrading profile "pctl-profile" from version "v0.1.0" to "v0.1.1"`,
				"✔ upgrade completed successfully",
			))
		})
	})

	When("using the latest flag with no updates", func() {
		It("returns an error", func() {
			gitRepoName := "my-git-repo"
			subname := "pctl-profile"
			profileDir := filepath.Join(temp, subname)
			args := []string{
				"add",
				"--name", subname,
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--config-map", configMapName,
				"nginx-catalog/weaveworks-nginx/v0.1.1",
			}

			Expect(pctl(args...)).To(ContainElements(
				"► generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.1",
				"✔ installation completed successfully",
			))
			By("upgrading a profile")
			args = []string{
				"upgrade",
				"--latest",
				profileDir,
			}
			Expect(pctlWithError(args...)).To(ConsistOf(
				`✗ no new versions available`,
			))
		})
	})

	When("merge conflicts occur", func() {
		It("informs the user of where the conflicts are", func() {
			By("installing a profile")
			gitRepoName := "my-git-repo"
			subname := "pctl-profile"
			args := []string{
				"add",
				"--name", subname,
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--config-map", configMapName,
				"nginx-catalog/weaveworks-nginx/v0.1.0",
			}

			Expect(pctl(args...)).To(ContainElements(
				"► generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0",
				"✔ installation completed successfully",
			))

			profileDir := filepath.Join(temp, subname)
			By("creating the artifacts")
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

			filename := filepath.Join(profileDir, "profile-installation.yaml")
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
    profile: weaveworks-nginx
    version: v0.1.0
  configMap: %s
  gitRepository:
    name: my-git-repo
    namespace: %s
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.0
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName, namespace)))

			By("manual editing the profile")
			deploymentFile := filepath.Join(profileDir, "artifacts/nginx-deployment/nginx/deployment/deployment.yaml")
			input, err := ioutil.ReadFile(deploymentFile)
			Expect(err).NotTo(HaveOccurred())

			//the latest version changes this to 3, checking it's currently 2 for sanity
			Expect(strings.Contains(string(input), "replicas: 2")).To(BeTrue())
			//modify the replicas to cause a merger conflict
			input = []byte(strings.Replace(string(input), "replicas: 2", "replicas: 10", -1))

			err = ioutil.WriteFile(deploymentFile, []byte(input), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("upgrading a profile")
			args = []string{
				"upgrade",
				profileDir,
				"v0.1.1",
			}
			Expect(pctlWithError(args...)).To(ConsistOf(
				`► upgrading profile "pctl-profile" from version "v0.1.0" to "v0.1.1"`,
				"✗ upgrade succeeded but merge conflicts have occurred, please resolve manually. Files containing conflicts:",
				fmt.Sprintf("- %s", deploymentFile),
			))

			By("updating the artifacts")
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

			content, err = ioutil.ReadFile(filename)
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
    profile: weaveworks-nginx
    version: v0.1.1
  configMap: %s
  gitRepository:
    name: my-git-repo
    namespace: %s
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.1
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName, namespace)))
		})
	})
})
