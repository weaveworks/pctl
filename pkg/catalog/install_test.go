package catalog_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	"github.com/weaveworks/pctl/pkg/git"
	gitfakes "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/profile"
)

var _ = Describe("Install", func() {
	var (
		fakeCatalogClient *fakes.FakeCatalogClient
		fakeGit           *gitfakes.FakeGit
		fakeScm           *gitfakes.FakeSCMClient
		tempDir           string
		httpBody          []byte
		cfg               catalog.InstallConfig
		fakeMakeArtifacts catalog.MakeArtifacts
		gitRepoNamespace  = "git-repo-namespace"
		gitRepoName       = "git-repo-name"
	)

	BeforeEach(func() {
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		fakeGit = new(gitfakes.FakeGit)
		fakeScm = new(gitfakes.FakeSCMClient)
		var err error
		tempDir, err = ioutil.TempDir("", "catalog-install")
		Expect(err).NotTo(HaveOccurred())
		httpBody = []byte(`
{
	"name": "nginx-1",
	"description": "nginx 1",
	"version": "v0.0.1",
	"catalog": "weaveworks (https://github.com/weaveworks/profiles)",
	"url": "https://github.com/weaveworks/nginx-profile",
	"prerequisites": ["Kubernetes 1.18+"],
	"maintainer": "WeaveWorks <gitops@weave.works>"
}
`)
		fakeCatalogClient.DoRequestReturns(httpBody, 200, nil)
		cfg = catalog.InstallConfig{
			ProfileConfig: catalog.ProfileConfig{
				CatalogName:   "nginx",
				GitRepository: gitRepoNamespace + "/" + gitRepoName,
				Namespace:     "default",
				ProfileBranch: "main",
				ProfileName:   "nginx-1",
				SubName:       "mysub",
				Version:       "v0.0.1",
			},
			Clients: catalog.Clients{
				CatalogClient: fakeCatalogClient,
				GitClient:     fakeGit,
			},
			Directory: tempDir,
		}
		fakeMakeArtifacts = func(sub profilesv1.ProfileSubscription, gitClient git.Git, rootDir, gitRepoNamespace string, gitRepoName string) ([]profile.Artifact, error) {
			return []profile.Artifact{
				{
					Objects: []runtime.Object{
						&kustomizev1.Kustomization{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "foo",
								Namespace: "default",
							},
							TypeMeta: metav1.TypeMeta{
								Kind:       "kustomize",
								APIVersion: "api",
							},
							Spec: kustomizev1.KustomizationSpec{
								Prune: true,
							},
						},
					},
					Name: "foo",
				},
			}, nil
		}
	})

	JustBeforeEach(func() {
		catalog.SetMakeArtifacts(fakeMakeArtifacts)
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
	})

	Describe("install", func() {
		It("generates the artifacts", func() {
			err := catalog.Install(cfg)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			profileDir := filepath.Join(tempDir, "nginx-1")
			err = filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			profileFile := filepath.Join(profileDir, "profile.yaml")
			artifactFile := filepath.Join(profileDir, "artifacts", "foo", "kustomize.yaml")
			Expect(files).To(ConsistOf(profileFile, artifactFile))

			Expect(hasCorrectFilePerms(profileFile)).To(BeTrue())
			Expect(hasCorrectFilePerms(artifactFile)).To(BeTrue())

			content, err := ioutil.ReadFile(profileFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  profile_catalog_description:
    catalog: nginx
    profile: nginx-1
    version: v0.0.1
  profileURL: https://github.com/weaveworks/nginx-profile
  version: nginx-1/v0.0.1
status: {}
`))

			content, err = ioutil.ReadFile(artifactFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: api
kind: kustomize
metadata:
  creationTimestamp: null
  name: foo
  namespace: default
spec:
  interval: 0s
  prune: true
  sourceRef:
    kind: ""
    name: ""
status: {}
`))
		})

		When("getting the artifacts fails", func() {
			BeforeEach(func() {
				fakeMakeArtifacts = func(sub profilesv1.ProfileSubscription, gitClient git.Git, rootDir, gitRepoNamespace string, gitRepoName string) ([]profile.Artifact, error) {
					return nil, fmt.Errorf("foo")
				}
			})

			It("errors", func() {
				err := catalog.Install(cfg)
				Expect(err).To(MatchError("failed to generate artifacts: foo"))
			})
		})

		When("creating the dir fails", func() {
			BeforeEach(func() {
				cfg.Directory = "/23123~@$~!@£$~1'24!"
			})

			It("errors", func() {
				err := catalog.Install(cfg)
				Expect(err).To(MatchError(ContainSubstring("failed to create directory")))
			})
		})

		When("a url is provided with branch and path", func() {
			It("generates a spec with url and branch and path", func() {
				cfg = catalog.InstallConfig{
					Clients: catalog.Clients{
						CatalogClient: fakeCatalogClient,
						GitClient:     fakeGit,
					},
					ProfileConfig: catalog.ProfileConfig{
						CatalogName:   "nginx",
						Namespace:     "default",
						Path:          "branch-nginx",
						ProfileBranch: "main",
						ProfileName:   "nginx-1",
						SubName:       "mysub",
						URL:           "https://github.com/weaveworks/profiles-examples",
					},
					Directory: tempDir,
				}
				err := catalog.Install(cfg)
				Expect(err).NotTo(HaveOccurred())

				var files []string
				profileDir := filepath.Join(tempDir, "nginx-1")
				err = filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
					files = append(files, path)
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				profileFile := filepath.Join(profileDir, "profile.yaml")
				artifactsDir := filepath.Join(profileDir, "artifacts")
				artifactsFooDir := filepath.Join(profileDir, "artifacts", "foo")
				artifactFile := filepath.Join(profileDir, "artifacts", "foo", "kustomize.yaml")
				Expect(files).To(ConsistOf(artifactsDir, artifactsFooDir, profileDir, profileFile, artifactFile))

				content, err := ioutil.ReadFile(profileFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileSubscription
metadata:
  creationTimestamp: null
  name: mysub
  namespace: default
spec:
  branch: main
  path: branch-nginx
  profileURL: https://github.com/weaveworks/profiles-examples
status: {}
`))

				content, err = ioutil.ReadFile(artifactFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: api
kind: kustomize
metadata:
  creationTimestamp: null
  name: foo
  namespace: default
spec:
  interval: 0s
  prune: true
  sourceRef:
    kind: ""
    name: ""
status: {}
`))
			})
		})

		When("a branch is provided which isn't domain compatible", func() {
			It("will not care because the name is sanitised", func() {
				cfg = catalog.InstallConfig{
					Clients: catalog.Clients{
						CatalogClient: fakeCatalogClient,
						GitClient:     fakeGit,
					},
					ProfileConfig: catalog.ProfileConfig{
						CatalogName:   "nginx",
						Namespace:     "default",
						Path:          "path",
						ProfileBranch: "not_domain_compatible",
						ProfileName:   "nginx-1",
						SubName:       "mysub",
						URL:           "https://github.com/weaveworks/profiles-examples",
					},
					Directory: tempDir,
				}
				err := catalog.Install(cfg)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("create-pr", func() {
		When("create-pr is set to true", func() {
			It("can create a PR if the generated values result in changes", func() {
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
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
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
				Expect(err).To(MatchError("failed to create branch: nope"))
			})
			It("handles add errors", func() {
				fakeGit.AddReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
				Expect(err).To(MatchError("failed to add changes: nope"))
			})
			It("handles commit errors", func() {
				fakeGit.CommitReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
				Expect(err).To(MatchError("failed to commit changes: nope"))
			})
			It("handles push errors", func() {
				fakeGit.PushReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
				Expect(err).To(MatchError("failed to push changes: nope"))
			})
			It("handles create pull request errors", func() {
				fakeScm.CreatePullRequestReturns(errors.New("nope"))
				err := catalog.CreatePullRequest(fakeScm, fakeGit)
				Expect(err).To(MatchError("failed to create pull request: nope"))
			})
		})
	})
})

func hasCorrectFilePerms(file string) bool {
	info, err := os.Stat(file)
	Expect(err).NotTo(HaveOccurred())
	return strconv.FormatUint(uint64(info.Mode().Perm()), 8) == strconv.FormatInt(0644, 8)
}
