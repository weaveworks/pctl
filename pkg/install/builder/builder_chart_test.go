package builder_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"
)

var helmReleaseTypeMeta = metav1.TypeMeta{
	Kind:       "HelmRelease",
	APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
}
var helmRepoTypeMeta = metav1.TypeMeta{
	Kind:       "HelmRepository",
	APIVersion: "source.toolkit.fluxcd.io/v1beta1",
}

var _ = Describe("Helm", func() {
	var configMapName = "my-configmap"
	BeforeEach(func() {
		chartDir := filepath.Join(gitDir, "weaveworks-nginx", "files")
		Expect(os.MkdirAll(chartDir, 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(chartDir, "file1"), []byte("foo"), 0755)).To(Succeed())

		artifacts = []artifact.Artifact{
			{
				Artifact: profilesv1.Artifact{
					Name: artifactName,
					Chart: &profilesv1.Chart{
						Path:          "files/",
						DefaultValues: "values",
					},
				},
				ProfileRepoKey:            repoKey,
				ProfilePath:               profilePath,
				ParentProfileArtifactName: "",
			},
		}
	})

	It("generates the helm resources and copies the chart into the directory", func() {
		installation.Spec.ConfigMap = configMapName
		err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
		Expect(err).NotTo(HaveOccurred())

		var files []string
		err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				files = append(files, strings.TrimPrefix(path, rootDir+"/"))
			}
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(ConsistOf(
			"artifacts/1/kustomization.yaml",
			"artifacts/1/kustomize-flux.yaml",
			"artifacts/1/helm-chart/kustomization.yaml",
			"artifacts/1/helm-chart/HelmRelease.yaml",
			"artifacts/1/helm-chart/ConfigMap.yaml",
			"artifacts/1/helm-chart/files/file1",
			"profile-installation.yaml",
		))

		By("generating the wrapper kustomization with healthcheks")
		kustomization := types.Kustomization{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/kustomization.yaml"), &kustomization)
		Expect(kustomization).To(Equal(types.Kustomization{
			Resources: []string{"kustomize-flux.yaml"},
		}))

		kustomize := kustomizev1.Kustomization{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/kustomize-flux.yaml"), &kustomize)
		Expect(kustomize).To(Equal(kustomizev1.Kustomization{
			TypeMeta: kustomizeTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
				Namespace: namespace,
			},
			Spec: kustomizev1.KustomizationSpec{
				Path: filepath.Join(rootDir, "artifacts/1/helm-chart"),
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind:      "GitRepository",
					Namespace: gitRepoNamespace,
					Name:      gitRepoName,
				},
				HealthChecks: []meta.NamespacedObjectKindReference{
					{
						APIVersion: helmv2.GroupVersion.String(),
						Kind:       helmv2.HelmReleaseKind,
						Name:       fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
						Namespace:  namespace,
					},
				},
				Interval:        metav1.Duration{Duration: 300000000000},
				Prune:           true,
				TargetNamespace: namespace,
			},
		}))

		By("creating the helm resources and values")
		kustomization = types.Kustomization{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/helm-chart/kustomization.yaml"), &kustomization)
		Expect(kustomization).To(Equal(types.Kustomization{
			Resources: []string{"HelmRelease.yaml"},
		}))

		configMap := corev1.ConfigMap{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/helm-chart/ConfigMap.yaml"), &configMap)
		Expect(configMap).To(Equal(corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-defaultvalues", installationName, artifactName),
				Namespace: namespace,
			},
			Data: map[string]string{
				"default-values.yaml": `values`,
			},
		},
		))

		helmRes := helmv2.HelmRelease{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/helm-chart/HelmRelease.yaml"), &helmRes)
		Expect(helmRes).To(Equal(helmv2.HelmRelease{
			TypeMeta: helmReleaseTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
				Namespace: namespace,
			},
			Spec: helmv2.HelmReleaseSpec{
				Chart: helmv2.HelmChartTemplate{
					Spec: helmv2.HelmChartTemplateSpec{
						Chart: filepath.Join(rootDir, "artifacts/1/helm-chart/files/"),
						SourceRef: helmv2.CrossNamespaceObjectReference{
							Kind:      "GitRepository",
							Name:      gitRepoName,
							Namespace: gitRepoNamespace,
						},
					},
				},
				ValuesFrom: []helmv2.ValuesReference{
					{
						Name:      fmt.Sprintf("%s-%s-defaultvalues", installationName, artifactName),
						Kind:      "ConfigMap",
						ValuesKey: "default-values.yaml",
					},
					{
						Kind:      "ConfigMap",
						Name:      configMapName,
						ValuesKey: artifactName,
					},
				},
			},
		}))
	})

	When("using a remote helm chart", func() {
		var (
			chartName    = "chart-name"
			chartURL     = "example.com"
			chartVersion = "v1.0.0"
		)
		BeforeEach(func() {
			artifacts = []artifact.Artifact{
				{
					Artifact: profilesv1.Artifact{
						Name: artifactName,
						Chart: &profilesv1.Chart{
							URL:     chartURL,
							Version: chartVersion,
							Name:    chartName,
						},
					},
					ProfileRepoKey:            repoKey,
					ProfilePath:               profilePath,
					ParentProfileArtifactName: "",
				},
			}
		})

		It("generates the helm resources and copies the chart into the directory", func() {
			err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, rootDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ConsistOf(
				"artifacts/1/kustomization.yaml",
				"artifacts/1/kustomize-flux.yaml",
				"artifacts/1/helm-chart/HelmRelease.yaml",
				"artifacts/1/helm-chart/HelmRepository.yaml",
				"profile-installation.yaml",
			))

			By("generating the wrapper kustomization with healthcheks")
			kustomization := types.Kustomization{}
			decodeFile(filepath.Join(rootDir, "artifacts/1/kustomization.yaml"), &kustomization)
			Expect(kustomization).To(Equal(types.Kustomization{
				Resources: []string{"kustomize-flux.yaml"},
			}))

			kustomize := kustomizev1.Kustomization{}
			decodeFile(filepath.Join(rootDir, "artifacts/1/kustomize-flux.yaml"), &kustomize)
			Expect(kustomize).To(Equal(kustomizev1.Kustomization{
				TypeMeta: kustomizeTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
					Namespace: namespace,
				},
				Spec: kustomizev1.KustomizationSpec{
					Path: filepath.Join(rootDir, "artifacts/1/helm-chart"),
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Kind:      "GitRepository",
						Namespace: gitRepoNamespace,
						Name:      gitRepoName,
					},
					HealthChecks: []meta.NamespacedObjectKindReference{
						{
							APIVersion: helmv2.GroupVersion.String(),
							Kind:       helmv2.HelmReleaseKind,
							Name:       fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
							Namespace:  namespace,
						},
					},
					Interval:        metav1.Duration{Duration: 300000000000},
					Prune:           true,
					TargetNamespace: namespace,
				},
			}))

			By("creating the helm resources")
			helmRes := helmv2.HelmRelease{}
			decodeFile(filepath.Join(rootDir, "artifacts/1/helm-chart/HelmRelease.yaml"), &helmRes)
			Expect(helmRes).To(Equal(helmv2.HelmRelease{
				TypeMeta: helmReleaseTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
					Namespace: namespace,
				},
				Spec: helmv2.HelmReleaseSpec{
					Chart: helmv2.HelmChartTemplate{
						Spec: helmv2.HelmChartTemplateSpec{
							Chart:   chartName,
							Version: chartVersion,
							SourceRef: helmv2.CrossNamespaceObjectReference{
								Kind:      "HelmRepository",
								Name:      fmt.Sprintf("%s-profiles-examples-%s", installationName, chartName),
								Namespace: namespace,
							},
						},
					},
				},
			}))

			helmRepo := sourcev1.HelmRepository{}
			decodeFile(filepath.Join(rootDir, "artifacts/1/helm-chart/HelmRepository.yaml"), &helmRepo)
			Expect(helmRepo).To(Equal(sourcev1.HelmRepository{
				TypeMeta: helmRepoTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-profiles-examples-%s", installationName, chartName),
					Namespace: namespace,
				},
				Spec: sourcev1.HelmRepositorySpec{
					URL: chartURL,
				},
			}))
		})
	})

	When("the gitrepository isn't set", func() {
		It("returns an error", func() {
			artifactBuilder.GitRepositoryName = ""
			err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
			Expect(err).To(MatchError("in case of local resources, the flux gitrepository object's details must be provided"))
		})
	})

	When("the repo hasn't been cloned", func() {
		It("returns an error", func() {
			artifacts[0].ProfileRepoKey = "dontexistlol"
			err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
			Expect(err).To(MatchError(ContainSubstring("could not find repo clone for \"dontexistlol\"")))
		})
	})

	When("copying the artifact fails", func() {
		It("returns an error", func() {
			artifacts[0].ProfilePath = "/tmp/i/dont/exist"
			err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
			Expect(err).To(MatchError(ContainSubstring("failed to copy files:")))
		})
	})
})
