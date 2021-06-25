package chart_test

import (
	"path/filepath"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/weaveworks/pctl/pkg/profile/artifact"
	"github.com/weaveworks/pctl/pkg/profile/chart"
)

var _ = Describe("Builder", func() {
	var (
		profileName            string
		profileURL             string
		profilePath            string
		partifact              profilesv1.Artifact
		pSub                   profilesv1.ProfileInstallation
		pDef                   profilesv1.ProfileDefinition
		rootDir                string
		gitRepositoryName      string
		gitRepositoryNamespace string
		profileName1           = "weaveworks-nginx"
		namespace              = "default"
		profileSubAPIVersion   = "weave.works/v1alpha1"
		profileSubKind         = "ProfileInstallation"
	)

	var (
		profileTypeMeta = metav1.TypeMeta{
			Kind:       profileSubKind,
			APIVersion: profileSubAPIVersion,
		}
	)
	BeforeEach(func() {
		profileName = "test-profile"
		profileURL = "https://github.com/weaveworks/profiles-examples"
		profilePath = "weaveworks-nginx"
		pSub = profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      profileName,
				Namespace: namespace,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL:    profileURL,
					Branch: "main",
					Path:   profilePath,
				},
				ConfigMap: "nginx-values",
			},
		}
		partifact = profilesv1.Artifact{
			Name: "dokuwiki",
			Chart: &profilesv1.Chart{
				URL:     "https://charts.bitnami.com/bitnami",
				Name:    "dokuwiki",
				Version: "11.1.6",
			},
		}
		pDef = profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: profileName1,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Profile",
				APIVersion: "packages.weave.works/profilesv1",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				ProfileDescription: profilesv1.ProfileDescription{
					Description: "foo",
				},
				Artifacts: []profilesv1.Artifact{partifact},
			},
		}
		rootDir = "root-dir"
		gitRepositoryName = "git-repository-name"
		gitRepositoryNamespace = "git-repository-namespace"
	})

	Context("Build", func() {
		When("a remote chart is configured", func() {
			It("creates an artifact from an install and a profile definition", func() {
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						GitRepositoryName:      gitRepositoryName,
						GitRepositoryNamespace: gitRepositoryNamespace,
						RootDir:                rootDir,
					},
				}
				artifacts, err := chartBuilder.Build(partifact, pSub, pDef)
				Expect(err).NotTo(HaveOccurred())
				helmRelease := &helmv2.HelmRelease{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HelmRelease",
						APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-weaveworks-nginx-dokuwiki",
						Namespace: "default",
					},
					Spec: helmv2.HelmReleaseSpec{
						ValuesFrom: []helmv2.ValuesReference{
							{
								Kind:      "ConfigMap",
								Name:      "nginx-values",
								ValuesKey: "dokuwiki",
							},
						},
						Chart: helmv2.HelmChartTemplate{
							Spec: helmv2.HelmChartTemplateSpec{
								Chart:   "dokuwiki",
								Version: "11.1.6",
								SourceRef: helmv2.CrossNamespaceObjectReference{
									Kind:      "HelmRepository",
									Name:      "test-profile-profiles-examples-dokuwiki",
									Namespace: "default",
								},
							},
						},
					},
				}
				helmRepository := &sourcev1.HelmRepository{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HelmRepository",
						APIVersion: "source.toolkit.fluxcd.io/v1beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-profiles-examples-dokuwiki",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						URL: "https://charts.bitnami.com/bitnami",
					},
				}
				expected := artifact.Artifact{
					Objects: []runtime.Object{helmRelease, helmRepository},
					Name:    "dokuwiki",
				}
				Expect(artifacts).To(ConsistOf(expected))
			})
		})

		When("a path based chart is configured", func() {
			BeforeEach(func() {

				partifact = profilesv1.Artifact{
					Name: "dokuwiki",
					Chart: &profilesv1.Chart{
						Path: "my/chart",
					},
				}
			})
			It("creates an artifact from an install and a profile definition", func() {
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						GitRepositoryName:      gitRepositoryName,
						GitRepositoryNamespace: gitRepositoryNamespace,
						RootDir:                rootDir,
					},
				}
				artifacts, err := chartBuilder.Build(partifact, pSub, pDef)
				Expect(err).NotTo(HaveOccurred())
				helmRelease := &helmv2.HelmRelease{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HelmRelease",
						APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-weaveworks-nginx-dokuwiki",
						Namespace: "default",
					},
					Spec: helmv2.HelmReleaseSpec{
						ValuesFrom: []helmv2.ValuesReference{
							{
								Kind:      "ConfigMap",
								Name:      "nginx-values",
								ValuesKey: "dokuwiki",
							},
						},
						Chart: helmv2.HelmChartTemplate{
							Spec: helmv2.HelmChartTemplateSpec{
								Chart: filepath.Join(rootDir, "artifacts", "dokuwiki", "my/chart"),
								SourceRef: helmv2.CrossNamespaceObjectReference{
									Kind:      "GitRepository",
									Name:      gitRepositoryName,
									Namespace: gitRepositoryNamespace,
								},
							},
						},
					},
				}
				Expect(artifacts).To(HaveLen(1))
				Expect(artifacts[0].Name).To(Equal("dokuwiki"))
				Expect(artifacts[0].RepoURL).To(Equal(profileURL))
				Expect(artifacts[0].SparseFolder).To(Equal(profileName1))
				Expect(artifacts[0].Branch).To(Equal("main"))
				Expect(artifacts[0].PathsToCopy).To(ConsistOf("my/chart"))
				Expect(*artifacts[0].Kustomize).To(Equal(types.Kustomization{
					Resources: []string{"HelmRelease.yaml"},
				}))
				Expect(artifacts[0].Objects).To(ConsistOf(helmRelease))
			})
		})

		When("git-repository-name and git-repository-namespace aren't defined", func() {
			It("returns an error", func() {
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						RootDir: rootDir,
					},
				}
				partifact = profilesv1.Artifact{
					Name: "local-artifact",
					Chart: &profilesv1.Chart{
						Path: "nginx/chart",
					},
				}
				pDef = profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: profileName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Profile",
						APIVersion: "packages.weave.works/profilesv1",
					},
					Spec: profilesv1.ProfileDefinitionSpec{
						ProfileDescription: profilesv1.ProfileDescription{
							Description: "foo",
						},
						Artifacts: []profilesv1.Artifact{partifact},
					},
				}
				_, err := chartBuilder.Build(partifact, pSub, pDef)
				Expect(err).To(MatchError("in case of local resources, the flux gitrepository object's details must be provided"))
			})
		})

		When("helmRepository and path", func() {
			It("errors", func() {

				a := profilesv1.Artifact{
					Name: "test",
					Chart: &profilesv1.Chart{
						Name:    "chart",
						Version: "v0.0.1",
						URL:     "https://github.com",
						Path:    "path",
					},
				}
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						RootDir: rootDir,
					},
				}
				_, err := chartBuilder.Build(a, pSub, pDef)
				Expect(err).To(MatchError(ContainSubstring("validation failed for artifact test: expected exactly one, got both: chart.path, chart.url")))
			})
		})
		When("chart and kustomize", func() {
			It("errors", func() {
				a := profilesv1.Artifact{
					Name: "test",
					Chart: &profilesv1.Chart{
						Name: "chart",
						URL:  "https://github.com",
					},
					Kustomize: &profilesv1.Kustomize{
						Path: "https://not.empty",
					},
				}
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						RootDir: rootDir,
					},
				}
				_, err := chartBuilder.Build(a, pSub, pDef)
				Expect(err).To(MatchError(ContainSubstring("validation failed for artifact test: expected exactly one, got both: chart, kustomize")))
			})
		})
		When("helmRepository and profile", func() {
			It("errors", func() {
				a := profilesv1.Artifact{
					Name: "test",
					Chart: &profilesv1.Chart{
						URL: "https://github.com",
					},
					Profile: &profilesv1.Profile{
						Source: &profilesv1.Source{
							URL:    "example.com",
							Branch: "branch",
						},
					},
				}
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						RootDir: rootDir,
					},
				}
				_, err := chartBuilder.Build(a, pSub, pDef)
				Expect(err).To(MatchError(ContainSubstring("validation failed for artifact test: expected exactly one, got both: chart, profile")))
			})
		})
		When("the helm chart has default values set", func() {
			It("will apply those values to the profile installation", func() {
				partifact = profilesv1.Artifact{
					Name: "dokuwiki",
					Chart: &profilesv1.Chart{
						URL:           "https://charts.bitnami.com/bitnami",
						Name:          "dokuwiki",
						Version:       "11.1.6",
						DefaultValues: `{"foo": "bar", "service": {"port": 1234}}`,
					},
				}
				pDef = profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: profileName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Profile",
						APIVersion: "packages.weave.works/profilesv1",
					},
					Spec: profilesv1.ProfileDefinitionSpec{
						ProfileDescription: profilesv1.ProfileDescription{
							Description: "foo",
						},
						Artifacts: []profilesv1.Artifact{partifact},
					},
				}
				chartBuilder := &chart.Builder{
					Config: chart.Config{
						GitRepositoryName:      gitRepositoryName,
						GitRepositoryNamespace: gitRepositoryNamespace,
						RootDir:                rootDir,
					},
				}
				artifacts, err := chartBuilder.Build(partifact, pSub, pDef)
				Expect(err).NotTo(HaveOccurred())
				configMap := &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-dokuwiki-defaultvalues",
						Namespace: "default",
					},
					Data: map[string]string{
						"default-values.yaml": `{"foo": "bar", "service": {"port": 1234}}`,
					},
				}
				helmRelease := &helmv2.HelmRelease{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HelmRelease",
						APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-weaveworks-nginx-dokuwiki",
						Namespace: "default",
					},
					Spec: helmv2.HelmReleaseSpec{
						Chart: helmv2.HelmChartTemplate{
							Spec: helmv2.HelmChartTemplateSpec{
								Chart:   "dokuwiki",
								Version: "11.1.6",
								SourceRef: helmv2.CrossNamespaceObjectReference{
									Kind:      "HelmRepository",
									Name:      "test-profile-profiles-examples-dokuwiki",
									Namespace: "default",
								},
							},
						},
						ValuesFrom: []helmv2.ValuesReference{
							{
								Name:      "test-profile-dokuwiki-defaultvalues",
								Kind:      "ConfigMap",
								ValuesKey: "default-values.yaml",
							},
							{
								Kind:      "ConfigMap",
								Name:      "nginx-values",
								ValuesKey: "dokuwiki",
							},
						},
					},
				}
				helmRepository := &sourcev1.HelmRepository{
					TypeMeta: metav1.TypeMeta{
						Kind:       "HelmRepository",
						APIVersion: "source.toolkit.fluxcd.io/v1beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-profiles-examples-dokuwiki",
						Namespace: "default",
					},
					Spec: sourcev1.HelmRepositorySpec{
						URL: "https://charts.bitnami.com/bitnami",
					},
				}
				expected := artifact.Artifact{
					Objects: []runtime.Object{configMap, helmRelease, helmRepository},
					Name:    "dokuwiki",
				}
				Expect(artifacts).To(ConsistOf(expected))
			})
		})
	})
})
