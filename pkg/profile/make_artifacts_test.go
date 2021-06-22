package profile_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile"
)

const (
	branch               = "main"
	gitRepoName          = "git-repo-name"
	gitRepoNamespace     = "git-repo-namespace"
	installationName     = "mySub"
	namespace            = "default"
	profileName1         = "weaveworks-nginx"
	profileName2         = "bitnami-nginx"
	profileSubAPIVersion = "weave.works/v1alpha1"
	profileSubKind       = "ProfileInstallation"
	profileURL           = "https://github.com/org/repo-name"
)

var (
	profileTypeMeta = metav1.TypeMeta{
		Kind:       profileSubKind,
		APIVersion: profileSubAPIVersion,
	}
)

var _ = Describe("MakeArtifactsFunc", func() {
	var (
		pSub          profilesv1.ProfileInstallation
		pDef          profilesv1.ProfileDefinition
		pNestedDef    profilesv1.ProfileDefinition
		pNestedDefURL = "https://github.com/org/repo-name-nested"
		rootDir       string
	)

	BeforeEach(func() {
		pSub = profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      installationName,
				Namespace: namespace,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL:    profileURL,
					Branch: branch,
					Path:   profileName1,
				},
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
				Artifacts: []profilesv1.Artifact{
					{
						Name: profileName2,
						Profile: &profilesv1.Profile{
							Source: &profilesv1.Source{
								URL: pNestedDefURL,
								Tag: "bitnami-nginx/v0.0.1",
							},
						},
					},
					{
						Name: "nginx-deployment",
						Kustomize: &profilesv1.Kustomize{
							Path: "nginx/deployment",
						},
					},
					{
						Name: "dokuwiki",
						Chart: &profilesv1.Chart{
							URL:     "https://charts.bitnami.com/bitnami",
							Name:    "dokuwiki",
							Version: "11.1.6",
						},
					},
				},
			},
		}

		pNestedDef = profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: profileName2,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Profile",
				APIVersion: "packages.weave.works.io/profilesv1",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				ProfileDescription: profilesv1.ProfileDescription{
					Description: "foo",
				},
				Artifacts: []profilesv1.Artifact{
					{
						Name: "chart",
						Chart: &profilesv1.Chart{
							Path: "nginx/chart",
						},
					},
				},
			},
		}
		profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
			if path == "weaveworks-nginx" {
				return pDef, nil
			}
			return pNestedDef, nil
		})
	})

	Context("makeArtifact", func() {
		It("generates the artifacts", func() {
			maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				RootDir:          rootDir,
				GitRepoNamespace: gitRepoNamespace,
				GitRepoName:      gitRepoName,
			})
			artifacts, err := profile.MakeArtifacts(maker, pSub)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifacts).To(HaveLen(3))

			nestedProfile := artifacts[0]
			Expect(nestedProfile.Name).To(Equal("bitnami-nginx/chart"))
			Expect(nestedProfile.RepoURL).To(Equal("https://github.com/org/repo-name-nested"))
			Expect(nestedProfile.PathsToCopy).To(Equal([]string{"nginx/chart"}))
			Expect(nestedProfile.SparseFolder).To(Equal("bitnami-nginx"))
			Expect(nestedProfile.Branch).To(Equal("bitnami-nginx/v0.0.1"))
			Expect(nestedProfile.Objects).To(HaveLen(1)) // we test the object's generation in their respective builder tests
			Expect(*nestedProfile.Kustomize).To(Equal(types.Kustomization{
				Resources: []string{"HelmRelease.yaml"},
			}))

			weaveworksNginx := artifacts[1]
			Expect(weaveworksNginx.Name).To(Equal("nginx-deployment"))
			Expect(weaveworksNginx.RepoURL).To(Equal("https://github.com/org/repo-name"))
			Expect(weaveworksNginx.PathsToCopy).To(Equal([]string{"nginx/deployment"}))
			Expect(weaveworksNginx.SparseFolder).To(Equal("weaveworks-nginx"))
			Expect(weaveworksNginx.Branch).To(Equal("main"))
			Expect(weaveworksNginx.Objects).To(HaveLen(1)) // we test the object's generation in their respective builder tests

			dokuWiki := artifacts[2]
			Expect(dokuWiki.Name).To(Equal("dokuwiki"))
			Expect(dokuWiki.Objects).To(HaveLen(2)) // we test the object's generation in their respective builder tests
		})

		When("fetching the nested profile definition fails", func() {
			It("returns an error", func() {
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
					return profilesv1.ProfileDefinition{}, fmt.Errorf("foo")
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("failed to get profile definition: foo")))
			})
		})
		When("profile artifact points to itself", func() {
			It("returns an error", func() {
				pDef = profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Profile",
						APIVersion: "packages.weave.works/profilesv1",
					},
					Spec: profilesv1.ProfileDefinitionSpec{
						ProfileDescription: profilesv1.ProfileDescription{
							Description: "foo",
						},
						Artifacts: []profilesv1.Artifact{
							{
								Name: "test",
								Profile: &profilesv1.Profile{
									Source: &profilesv1.Source{
										URL: pNestedDefURL,
										Tag: "bitnami-nginx/v0.0.1",
									},
								},
							},
						},
					},
				}
				profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
					return pDef, nil
				})
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("recursive artifact detected: profile https://github.com/org/repo-name-nested on branch  contains an artifact that points recursively back at itself")))
			})
		})
		When("the Kind of artifact is unknown", func() {
			It("returns an error", func() {
				pDef = profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Profile",
						APIVersion: "packages.weave.works/profilesv1",
					},
					Spec: profilesv1.ProfileDefinitionSpec{
						ProfileDescription: profilesv1.ProfileDescription{
							Description: "foo",
						},
						Artifacts: []profilesv1.Artifact{
							{
								Name: "test",
							},
						},
					},
				}
				profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
					return pDef, nil
				})
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("no artifact set")))
			})
		})
		When("the nested profile is invalid", func() {
			BeforeEach(func() {
				pNestedDef.Spec.Artifacts[0] = profilesv1.Artifact{}
			})
			It("returns an error", func() {
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("no artifact set")))
			})
		})
	})
})
