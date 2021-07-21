package catalog_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	gitfakes "github.com/weaveworks/pctl/pkg/git/fakes"
	installerfake "github.com/weaveworks/pctl/pkg/install/fakes"
)

var _ = Describe("Install", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
		fakeGit           *gitfakes.FakeGit
		fakeScm           *gitfakes.FakeSCMClient
		tempDir           string
		httpBody          []byte
		cfg               catalog.InstallConfig
		fakeInstaller     *installerfake.FakeProfileInstaller
		manager           catalog.Manager
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		fakeGit = new(gitfakes.FakeGit)
		fakeScm = new(gitfakes.FakeSCMClient)
		fakeInstaller = new(installerfake.FakeProfileInstaller)
		var err error
		tempDir, err = ioutil.TempDir("", "catalog-install")
		Expect(err).NotTo(HaveOccurred())
		httpBody = []byte(`{"item":
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "v0.0.1",
	"tag": "nginx-1/v0.0.1",
	"catalogSource": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}}
`)
		fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
		cfg = catalog.InstallConfig{
			Profile: catalog.Profile{
				ProfileConfig: catalog.ProfileConfig{
					CatalogName:   "nginx",
					Namespace:     "default",
					ProfileBranch: "main",
					ProfileName:   "nginx-1",
					SubName:       "mysub",
					Version:       "v0.0.1",
					ConfigMap:     "config-map",
				},
				GitRepoConfig: catalog.GitRepoConfig{
					Name:      "git-name",
					Namespace: "git-namespace",
				},
			},
			Clients: catalog.Clients{
				CatalogClient: fakeCatalogClient,
				Installer:     fakeInstaller,
			},
		}
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
	})

	Describe("install", func() {
		When("installing from a catalog entry", func() {
			It("generates the artifacts", func() {
				err := manager.Install(cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeInstaller.InstallCallCount()).To(Equal(1))
				arg := fakeInstaller.InstallArgsForCall(0)
				Expect(arg).To(Equal(profilesv1.ProfileInstallation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileInstallation",
						APIVersion: "weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mysub",
						Namespace: "default",
					},
					Spec: profilesv1.ProfileInstallationSpec{
						ConfigMap: "config-map",
						Source: &profilesv1.Source{
							URL:    "https://github.com/weaveworks/nginx-profile",
							Branch: "",
							Path:   "nginx-1",
							Tag:    "nginx-1/v0.0.1",
						},
						Catalog: &profilesv1.Catalog{
							Version: "v0.0.1",
							Catalog: "nginx",
							Profile: "nginx-1",
						},
						GitRepository: &profilesv1.GitRepository{
							Name:      "git-name",
							Namespace: "git-namespace",
						},
					},
				}))
			})
		})

		When("installing from a url", func() {
			BeforeEach(func() {
				cfg = catalog.InstallConfig{
					Profile: catalog.Profile{
						ProfileConfig: catalog.ProfileConfig{
							Namespace:     "default",
							ProfileBranch: "main",
							Path:          "nginx-1",
							URL:           "https://github.com/weaveworks/nginx-profile",
							SubName:       "mysub",
							Version:       "v0.0.1",
							ConfigMap:     "config-map",
						},
						GitRepoConfig: catalog.GitRepoConfig{
							Name:      "git-name",
							Namespace: "git-namespace",
						},
					},
					Clients: catalog.Clients{
						CatalogClient: fakeCatalogClient,
						Installer:     fakeInstaller,
					},
				}
			})
			It("generates the artifacts", func() {
				err := manager.Install(cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeInstaller.InstallCallCount()).To(Equal(1))
				arg := fakeInstaller.InstallArgsForCall(0)
				Expect(arg).To(Equal(profilesv1.ProfileInstallation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileInstallation",
						APIVersion: "weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mysub",
						Namespace: "default",
					},
					Spec: profilesv1.ProfileInstallationSpec{
						ConfigMap: "config-map",
						Source: &profilesv1.Source{
							URL:    "https://github.com/weaveworks/nginx-profile",
							Branch: "main",
							Path:   "nginx-1",
						},
						GitRepository: &profilesv1.GitRepository{
							Name:      "git-name",
							Namespace: "git-namespace",
						},
					},
				}))
			})
		})

		When("getting the artifacts fails", func() {
			It("errors", func() {
				fakeInstaller.InstallReturns(fmt.Errorf("foo"))
				err := manager.Install(cfg)
				Expect(err).To(MatchError("failed to make artifacts: foo"))
			})
		})

		When("a branch is provided which isn't domain compatible", func() {
			It("will not care because the name is sanitised", func() {
				fakeInstaller.InstallReturns(nil)
				cfg = catalog.InstallConfig{
					Clients: catalog.Clients{
						CatalogClient: fakeCatalogClient,
						Installer:     fakeInstaller,
					},
					Profile: catalog.Profile{
						ProfileConfig: catalog.ProfileConfig{
							CatalogName:   "nginx",
							Namespace:     "default",
							Path:          "path",
							ProfileBranch: "not_domain_compatible",
							ProfileName:   "nginx-1",
							SubName:       "mysub",
							URL:           "https://github.com/weaveworks/profiles-examples",
						},
					},
				}
				err := manager.Install(cfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("create-pr", func() {
		When("create-pr is set to true", func() {
			It("can create a PR if the generated values result in changes", func() {
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeGit.CreateBranchCallCount()).To(Equal(1))
				Expect(fakeGit.AddCallCount()).To(Equal(1))
				Expect(fakeGit.CommitCallCount()).To(Equal(1))
				Expect(fakeGit.PushCallCount()).To(Equal(1))
				Expect(fakeScm.CreatePullRequestCallCount()).To(Equal(1))
			})
		})
		When("create-pr is set to true but something goes wrong", func() {
			It("handles create branch errors", func() {
				fakeGit.CreateBranchReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).To(MatchError("failed to create branch: nope"))
			})
			It("handles add errors", func() {
				fakeGit.AddReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).To(MatchError("failed to add changes: nope"))
			})
			It("handles commit errors", func() {
				fakeGit.CommitReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).To(MatchError("failed to commit changes: nope"))
			})
			It("handles push errors", func() {
				fakeGit.PushReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).To(MatchError("failed to push changes: nope"))
			})
			It("handles create pull request errors", func() {
				fakeScm.CreatePullRequestReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit, "branch")
				Expect(err).To(MatchError("failed to create pull request: nope"))
			})
		})
	})
})
