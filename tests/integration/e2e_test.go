package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	temp          string
	namespace     string
	configMapName string
)

var _ = Describe("install and upgrade", func() {
	BeforeEach(func() {
		var err error
		namespace = uuid.New().String()
		temp, err = ioutil.TempDir("", "pctl_test_install_upgrade")
		Expect(err).ToNot(HaveOccurred())
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
		_ = os.RemoveAll(temp)
		nsp := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_ = kClient.Delete(context.Background(), &nsp)

	})

	It("works", func() {
		By("installing a profile")
		gitRepoName := "my-git-repo"
		cmd := exec.Command(
			binaryPath,
			"install",
			"--git-repository",
			fmt.Sprintf("%s/%s", namespace, gitRepoName),
			"--namespace", namespace,
			"--config-map", configMapName,
			"nginx-catalog/weaveworks-nginx/v0.1.0")

		cmd.Dir = temp
		output, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed: %s", string(output)))
		Expect(string(output)).To(ContainSubstring("generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0"))
		Expect(string(output)).To(ContainSubstring("installation completed successfully"))

		var files []string
		profileDir := filepath.Join(temp, "weaveworks-nginx")
		err = filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				files = append(files, strings.TrimPrefix(path, profileDir+"/"))
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
			"artifacts/nginx-chart/helm-chart/helm-chart/HelmRelease.yaml",
			"artifacts/nginx-chart/helm-chart/helm-chart/HelmRepository.yaml",
			"artifacts/nginx-chart/helm-chart/kustomization.yaml",
			"artifacts/nginx-chart/helm-chart/kustomize-flux.yaml",
			"artifacts/nested-profile/nginx-server/helm-chart/helm-chart/HelmRelease.yaml",
			"artifacts/nested-profile/nginx-server/helm-chart/kustomization.yaml",
			"artifacts/nested-profile/nginx-server/helm-chart/kustomize-flux.yaml",
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
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.0
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName)))

		By("manual editing the profile")
		deploymentFile := filepath.Join(profileDir, "artifacts/nginx-deployment/nginx/deployment/deployment.yaml")
		input, err := ioutil.ReadFile(deploymentFile)
		Expect(err).NotTo(HaveOccurred())

		//the latest version changes this to 3, checking its curerntly 2 for sanity
		Expect(strings.Contains(string(input), "replicas: 2")).To(BeTrue())
		//sanity check its currently set to 90
		Expect(strings.Contains(string(input), "containerPort: 80")).To(BeTrue())
		//modify the containerPort to 30
		input = []byte(strings.Replace(string(input), "containerPort: 80", "containerPort: 30", -1))

		err = ioutil.WriteFile(deploymentFile, []byte(input), 0644)
		Expect(err).NotTo(HaveOccurred())

		By("upgrading a profile")
		cmd = exec.Command(
			binaryPath,
			"upgrade",
			"--git-repository",
			fmt.Sprintf("%s/%s", namespace, gitRepoName),
			profileDir,
			"v0.1.1")
		cmd.Dir = temp
		output, err = cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed: %s", string(output)))
		Expect(string(output)).To(ContainSubstring(`upgrading profile "pctl-profile" from version "v0.1.0" to "v0.1.1"`))
		Expect(string(output)).To(ContainSubstring("upgrade completed successfully"))

		By("updating the artifacts")
		Expect(files).To(ContainElements(
			"profile-installation.yaml",
			"artifacts/nginx-deployment/kustomization.yaml",
			"artifacts/nginx-deployment/kustomize-flux.yaml",
			"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
			"artifacts/nginx-chart/helm-chart/helm-chart/HelmRelease.yaml",
			"artifacts/nginx-chart/helm-chart/helm-chart/HelmRepository.yaml",
			"artifacts/nginx-chart/helm-chart/kustomize-flux.yaml",
			"artifacts/nginx-chart/helm-chart/kustomization.yaml",
			"artifacts/nested-profile/nginx-server/helm-chart/helm-chart/HelmRelease.yaml",
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
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.1
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName)))

		By("user changes and upstream changes being applied")
		input, err = ioutil.ReadFile(deploymentFile)
		Expect(err).NotTo(HaveOccurred())

		//the latest version changes this to 3
		Expect(strings.Contains(string(input), "replicas: 3")).To(BeTrue())
		//manual changes should be preserved
		Expect(strings.Contains(string(input), "containerPort: 30")).To(BeTrue())
	})

	When("merge conflicts occur", func() {
		It("informs the user of where the conflicts are", func() {
			By("installing a profile")
			gitRepoName := "my-git-repo"
			cmd := exec.Command(
				binaryPath,
				"install",
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				"--namespace", namespace,
				"--config-map", configMapName,
				"nginx-catalog/weaveworks-nginx/v0.1.0")

			cmd.Dir = temp
			output, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pctl install failed: %s", string(output)))
			Expect(string(output)).To(ContainSubstring("generating profile installation from source: catalog entry nginx-catalog/weaveworks-nginx/v0.1.0"))
			Expect(string(output)).To(ContainSubstring("installation completed successfully"))

			var files []string
			profileDir := filepath.Join(temp, "weaveworks-nginx")
			err = filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, profileDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("creating the artifacts")
			Expect(files).To(ContainElements(
				"profile-installation.yaml",
				"artifacts/nginx-deployment/kustomization.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/HelmRelease.yaml",
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
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.0
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName)))

			By("manual editing the profile")
			deploymentFile := filepath.Join(profileDir, "artifacts/nginx-deployment/nginx/deployment/deployment.yaml")
			input, err := ioutil.ReadFile(deploymentFile)
			Expect(err).NotTo(HaveOccurred())

			//the latest version changes this to 3, checking its curerntly 2 for sanity
			Expect(strings.Contains(string(input), "replicas: 2")).To(BeTrue())
			//modify the replicas to cause a merger conflict
			input = []byte(strings.Replace(string(input), "replicas: 2", "replicas: 10", -1))

			err = ioutil.WriteFile(deploymentFile, []byte(input), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("upgrading a profile")
			cmd = exec.Command(
				binaryPath,
				"upgrade",
				"--git-repository",
				fmt.Sprintf("%s/%s", namespace, gitRepoName),
				profileDir,
				"v0.1.1")
			cmd.Dir = temp
			output, err = cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			Expect(strings.Split(string(output), "\n")).To(
				ConsistOf(
					`upgrading profile "pctl-profile" from version "v0.1.0" to "v0.1.1"`,
					"upgrade succeeded but merge conflict have occured, please resolve manually. Files containing conflicts:",
					fmt.Sprintf("- %s", deploymentFile),
					"",
				),
			)

			By("updating the artifacts")
			Expect(files).To(ContainElements(
				"profile-installation.yaml",
				"artifacts/nginx-deployment/kustomization.yaml",
				"artifacts/nginx-deployment/nginx/deployment/deployment.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRelease.yaml",
				"artifacts/nginx-chart/helm-chart/HelmRepository.yaml",
				"artifacts/nested-profile/nginx-server/helm-chart/HelmRelease.yaml",
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
  source:
    path: weaveworks-nginx
    tag: weaveworks-nginx/v0.1.1
    url: https://github.com/weaveworks/profiles-examples
status: {}
`, namespace, configMapName)))
		})
	})
})
