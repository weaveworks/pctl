package profile_test

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/weaveworks/pctl/pkg/profile"
)

var _ = Describe("KustomizeBuilder", func() {
	var (
		profileName            string
		profileURL             string
		profilePath            string
		artifact               profilesv1.Artifact
		pSub                   profilesv1.ProfileInstallation
		pDef                   profilesv1.ProfileDefinition
		rootDir                string
		gitRepositoryName      string
		gitRepositoryNamespace string
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
					URL:  profileURL,
					Tag:  "weaveworks-nginx/v0.0.1",
					Path: profilePath,
				},
			},
		}
		artifact = profilesv1.Artifact{
			Name: "kustomize",
			Kustomize: &profilesv1.Kustomize{
				Path: "nginx/deployment",
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
				Artifacts: []profilesv1.Artifact{artifact},
			},
		}
		rootDir = "root-dir"
		gitRepositoryName = "git-repository-name"
		gitRepositoryNamespace = "git-repository-namespace"
	})

	Context("Build", func() {
		It("creates an artifact from an install and a profile definition", func() {
			builder := &profile.KustomizeBuilder{
				BuilderConfig: profile.BuilderConfig{
					GitRepositoryName:      gitRepositoryName,
					GitRepositoryNamespace: gitRepositoryNamespace,
					RootDir:                rootDir,
				},
			}
			artifacts, err := builder.Build(artifact, pSub, pDef)
			Expect(err).NotTo(HaveOccurred())
			kustomization := &kustomizev1.Kustomization{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Kustomization",
					APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-profile-weaveworks-nginx-kustomize",
					Namespace: "default",
				},
				Spec: kustomizev1.KustomizationSpec{
					Path: "root-dir/artifacts/kustomize/nginx/deployment",
					SourceRef: kustomizev1.CrossNamespaceSourceReference{
						Kind:      "GitRepository",
						Namespace: gitRepositoryNamespace,
						Name:      gitRepositoryName,
					},
					Interval:        metav1.Duration{Duration: 300000000000},
					Prune:           true,
					TargetNamespace: "default",
				},
			}
			expected := profile.Artifact{
				Objects:      []runtime.Object{kustomization},
				Name:         "kustomize",
				RepoURL:      "https://github.com/weaveworks/profiles-examples",
				PathsToCopy:  []string{"nginx/deployment"},
				SparseFolder: "weaveworks-nginx",
				Branch:       "weaveworks-nginx/v0.0.1",
			}
			Expect(artifacts).To(ConsistOf(expected))
		})
		When("branch is defined instead of tag", func() {
			It("will use the branch definition", func() {
				pSub = profilesv1.ProfileInstallation{
					TypeMeta: profileTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						Name:      profileName,
						Namespace: namespace,
					},
					Spec: profilesv1.ProfileInstallationSpec{
						Source: &profilesv1.Source{
							URL:    profileURL,
							Branch: "custom-branch",
							Path:   profilePath,
						},
					},
				}
				builder := &profile.KustomizeBuilder{
					BuilderConfig: profile.BuilderConfig{
						GitRepositoryName:      gitRepositoryName,
						GitRepositoryNamespace: gitRepositoryNamespace,
						RootDir:                rootDir,
					},
				}
				artifacts, err := builder.Build(artifact, pSub, pDef)
				Expect(err).NotTo(HaveOccurred())
				kustomization := &kustomizev1.Kustomization{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Kustomization",
						APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-profile-weaveworks-nginx-kustomize",
						Namespace: "default",
					},
					Spec: kustomizev1.KustomizationSpec{
						Path: "root-dir/artifacts/kustomize/nginx/deployment",
						SourceRef: kustomizev1.CrossNamespaceSourceReference{
							Kind:      "GitRepository",
							Namespace: gitRepositoryNamespace,
							Name:      gitRepositoryName,
						},
						Interval:        metav1.Duration{Duration: 300000000000},
						Prune:           true,
						TargetNamespace: "default",
					},
				}
				expected := profile.Artifact{
					Objects:      []runtime.Object{kustomization},
					Name:         "kustomize",
					RepoURL:      "https://github.com/weaveworks/profiles-examples",
					PathsToCopy:  []string{"nginx/deployment"},
					SparseFolder: "weaveworks-nginx",
					Branch:       "custom-branch",
				}
				Expect(artifacts).To(ConsistOf(expected))
			})
		})
		When("git-repository-name and git-repository-namespace aren't defined", func() {
			It("returns an error", func() {
				builder := &profile.KustomizeBuilder{
					BuilderConfig: profile.BuilderConfig{
						RootDir: rootDir,
					},
				}
				artifact = profilesv1.Artifact{
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
						Artifacts: []profilesv1.Artifact{artifact},
					},
				}
				_, err := builder.Build(artifact, pSub, pDef)
				Expect(err).To(MatchError("in case of local resources, the flux gitrepository object's details must be provided"))
			})
		})
	})
})
