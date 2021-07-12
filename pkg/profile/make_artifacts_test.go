package profile_test

import (
	"errors"
	"path/filepath"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/dependency"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"

	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/profile"
)

const (
	branch               = "main"
	gitRepoName          = "git-repo-name"
	gitRepoNamespace     = "git-repo-namespace"
	installationName     = "mySub"
	namespace            = "default"
	profileName1         = "weaveworks-nginx"
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
		pSub    profilesv1.ProfileInstallation
		rootDir string
		fakeGit *fakegit.FakeGit
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
		// Make clone give back specific profile.yaml files.
		fakeGit = &fakegit.FakeGit{}
		fakeGit.CloneStub = func(repo string, branch string, loc string) error {
			from := filepath.Join("testdata", "simple_with_nested")
			err := copy.Copy(from, loc)
			Expect(err).NotTo(HaveOccurred())
			return nil
		}
	})

	Context("makeArtifact", func() {
		It("generates the artifacts", func() {
			maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				GitClient:        fakeGit,
				RootDir:          rootDir,
				GitRepoNamespace: gitRepoNamespace,
				GitRepoName:      gitRepoName,
			})
			artifacts, err := profile.MakeArtifacts(maker, pSub)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifacts).To(HaveLen(3))

			nestedProfile := artifacts[0]
			Expect(nestedProfile.Name).To(Equal("nested-profile/nginx-server"))
			Expect(nestedProfile.RepoURL).To(Equal("https://github.com/weaveworks/profiles-examples"))
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

			nginxChart := artifacts[2]
			Expect(nginxChart.Name).To(Equal("nginx-chart"))
			Expect(nginxChart.Objects).To(HaveLen(3)) // we test the object's generation in their respective builder tests
		})

		When("a dependsOn artifact defined", func() {
			It("calls the builder with proper dependencies passed in", func() {
				fakeGit.CloneStub = func(repo string, branch string, loc string) error {
					from := filepath.Join("testdata", "dependson")
					err := copy.Copy(from, loc)
					Expect(err).NotTo(HaveOccurred())
					return nil
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				artifacts, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).NotTo(HaveOccurred())
				// we test the generated artifacts consistency in the builders, here we test if
				// depends on was generated correctly.
				Expect(len(artifacts)).To(Equal(4))

				dep1 := artifacts[1]
				helmRelease, ok := dep1.Objects[0].(*helmv2.HelmRelease)
				Expect(ok).To(BeTrue())
				Expect(helmRelease.Spec.DependsOn).To(ConsistOf(dependency.CrossNamespaceDependencyReference{
					Name:      "mySub-weaveworks-nginx-dependon",
					Namespace: "default",
				}))
				dep2 := artifacts[3]
				kustomization, ok := dep2.Objects[0].(*kustomizev1.Kustomization)
				Expect(ok).To(BeTrue(), "dep2.Objects's second item was not a kustomization object")
				Expect(kustomization.Spec.DependsOn).To(ConsistOf(dependency.CrossNamespaceDependencyReference{
					Name:      "mySub-weaveworks-nginx-dependon2",
					Namespace: "default",
				}))
			})
		})
		When("a dependsOn artifact is not in the list of artifacts", func() {
			It("returns an error", func() {
				fakeGit.CloneStub = func(repo string, branch string, loc string) error {
					from := filepath.Join("testdata", "dependson_missing")
					err := copy.Copy(from, loc)
					Expect(err).NotTo(HaveOccurred())
					return nil
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError("nginx-chart's depending artifact dependon not found in the list of artifacts"))
			})
		})
		When("fetching the nested profile definition fails", func() {
			It("returns an error", func() {
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				fakeGit.CloneReturns(errors.New("nope"))
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("failed to get profile definition: failed to clone the repo: nope")))
			})
		})
		When("fetching the first profile definition fails", func() {
			It("returns an error", func() {
				fakeGit := &fakegit.FakeGit{}
				fakeGit.CloneReturns(errors.New("nope"))
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("failed to get profile definition: failed to clone the repo: nope")))
			})
		})
		When("profile artifact points to itself", func() {
			It("returns an error", func() {
				fakeGit.CloneStub = func(repo string, branch string, loc string) error {
					from := filepath.Join("testdata", "nested_with_self_link")
					err := copy.Copy(from, loc)
					Expect(err).NotTo(HaveOccurred())
					return nil
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := profile.MakeArtifacts(maker, pSub)
				Expect(err).To(MatchError(ContainSubstring("recursive artifact detected: profile https://github.com/weaveworks/profiles-examples on branch  contains an artifact that points recursively back at itself")))
			})
		})
		When("the Kind of artifact is unknown", func() {
			It("returns an error", func() {
				fakeGit.CloneStub = func(repo string, branch string, loc string) error {
					from := filepath.Join("testdata", "unknown_artifact_kind")
					err := copy.Copy(from, loc)
					Expect(err).NotTo(HaveOccurred())
					return nil
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGit,
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
