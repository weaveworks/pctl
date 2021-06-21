package chart_test

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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
				Values: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicaCount": 3,"service":{"port":8081}}`),
				},
				ValuesFrom: []helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
					},
				},
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
					Values: &apiextensionsv1.JSON{
						Raw: []byte(`{"replicaCount": 3,"service":{"port":8081}}`),
					},
					ValuesFrom: []helmv2.ValuesReference{
						{
							Name:     "nginx-values",
							Kind:     "Secret",
							Optional: true,
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
	})
})
