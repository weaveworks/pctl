package upgrade_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/catalog/fakes"
	"github.com/weaveworks/pctl/pkg/upgrade"
	repofakes "github.com/weaveworks/pctl/pkg/upgrade/repo/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Upgrade", func() {
	var (
		fakeCatalogClient  *fakes.FakeCatalogClient
		fakeCatalogManager *fakes.FakeCatalogManager
		fakeRepoManager    *repofakes.FakeRepoManager
		cfg                upgrade.UpgradeConfig
		version            string
		profileDir         string
		workingDir         string
		copierCallCount    int
		copierArgs         [][]string
	)

	BeforeEach(func() {
		var err error
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		fakeCatalogManager = new(fakes.FakeCatalogManager)
		fakeRepoManager = new(repofakes.FakeRepoManager)
		version = "v0.1.1"
		profileDir, err = ioutil.TempDir("", "upgrade-test")
		Expect(err).NotTo(HaveOccurred())
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Mkdir(filepath.Join(workingDir, ".git/"), 0755)).To(Succeed())
		installation := profilesv1.ProfileInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pctl-installation",
				Namespace: "default",
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Catalog: &profilesv1.Catalog{
					Version: "v0.1.0",
					Profile: "my-profile",
					Catalog: "my-catalog",
				},
				ConfigMap: "my-config-map",
				GitRepository: &profilesv1.GitRepository{
					Name:      "foo",
					Namespace: "bar",
				},
			},
		}
		bytes, err := yaml.Marshal(installation)
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(filepath.Join(profileDir, "profile-installation.yaml"), bytes, 0755)).To(Succeed())
		fakeCatalogManager.ShowReturnsOnCall(0, profilesv1.ProfileCatalogEntry{
			Tag:           "v0.1.1",
			CatalogSource: "my-catalog",
			Name:          "my-profile2",
		}, nil)

		cfg = upgrade.UpgradeConfig{
			ProfileDir:     profileDir,
			Version:        version,
			CatalogClient:  fakeCatalogClient,
			CatalogManager: fakeCatalogManager,
			RepoManager:    fakeRepoManager,
			WorkingDir:     workingDir,
		}

		copierCallCount = 0
		copierArgs = nil
		upgrade.SetCopier(func(src, dst string) error {
			copierCallCount++
			copierArgs = append(copierArgs, []string{src, dst})
			return nil
		})

	})

	AfterEach(func() {
		_ = os.RemoveAll(profileDir)
		_ = os.RemoveAll(workingDir)
	})

	It("Upgrades the profile installation", func() {
		err := upgrade.Upgrade(cfg)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeCatalogManager.ShowCallCount()).To(Equal(1))
		_, catalogName, profileName, desiredVersion := fakeCatalogManager.ShowArgsForCall(0)
		Expect(catalogName).To(Equal("my-catalog"))
		Expect(profileName).To(Equal("my-profile"))
		Expect(desiredVersion).To(Equal("v0.1.1"))

		Expect(fakeRepoManager.CreateRepoWithContentCallCount()).To(Equal(1))

		By("installing the original profile in the repo")
		Expect(fakeCatalogManager.InstallCallCount()).To(Equal(0))
		createRepoWriteContentsFunc := fakeRepoManager.CreateRepoWithContentArgsForCall(0)
		Expect(createRepoWriteContentsFunc()).To(Succeed())
		Expect(fakeCatalogManager.InstallCallCount()).To(Equal(1))
		Expect(fakeCatalogManager.InstallArgsForCall(0).Profile).To(Equal(catalog.Profile{
			ProfileConfig: catalog.ProfileConfig{
				CatalogName: "my-catalog",
				ProfileName: "my-profile",
				Version:     "v0.1.0",
				ConfigMap:   "my-config-map",
			},
			GitRepoConfig: catalog.GitRepoConfig{
				Name:      "foo",
				Namespace: "bar",
			},
		}))

		By("copying the user changes into the user-changes branch")
		Expect(fakeRepoManager.CreateBranchWithContentFromMainCallCount()).To(Equal(2))
		branch, writeContentFunc := fakeRepoManager.CreateBranchWithContentFromMainArgsForCall(0)
		Expect(branch).To(Equal("user-changes"))
		Expect(writeContentFunc()).To(Succeed())
		Expect(copierCallCount).To(Equal(2))
		Expect(copierArgs[1]).To(ConsistOf(profileDir, workingDir))

		By("install the updated profile into the updates branch")
		Expect(fakeCatalogManager.InstallCallCount()).To(Equal(1))
		branch, writeContentFunc = fakeRepoManager.CreateBranchWithContentFromMainArgsForCall(1)
		Expect(branch).To(Equal("update-changes"))
		Expect(writeContentFunc()).To(Succeed())
		Expect(fakeCatalogManager.InstallCallCount()).To(Equal(2))
		Expect(fakeCatalogManager.InstallArgsForCall(1).Profile).To(Equal(catalog.Profile{
			ProfileConfig: catalog.ProfileConfig{
				CatalogName: "my-catalog",
				ProfileName: "my-profile",
				Version:     "v0.1.1",
				ConfigMap:   "my-config-map",
			},
			GitRepoConfig: catalog.GitRepoConfig{
				Name:      "foo",
				Namespace: "bar",
			},
		}))

		By("merging the two and copying the outcome into the profile installation")
		Expect(fakeRepoManager.MergeBranchesCallCount()).To(Equal(1))
		branch1, branch2 := fakeRepoManager.MergeBranchesArgsForCall(0)
		Expect(branch1).To(Equal("update-changes"))
		Expect(branch2).To(Equal("user-changes"))
		Expect(copierArgs[0]).To(ConsistOf(workingDir, profileDir))
	})

	When("merge conflicts occur", func() {
		BeforeEach(func() {
			fakeRepoManager.MergeBranchesReturns([]string{"foo/bar"}, nil)
		})

		It("returns a list of files that contain conflicts", func() {
			err := upgrade.Upgrade(cfg)
			expectedErrMsg := fmt.Sprintf("upgrade succeeded but merge conflicts have occurred, please resolve manually. Files containing conflicts:\n- %s", filepath.Join(profileDir, "foo/bar"))
			Expect(err).To(MatchError(expectedErrMsg))
		})
	})

	When("the profile installation doesn't exist", func() {
		BeforeEach(func() {
			cfg.ProfileDir = "/tmp/totally/dont/exist"
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to read profile installation:")))
		})
	})

	When("the profile installation file isn't valid", func() {
		BeforeEach(func() {
			Expect(os.Remove(filepath.Join(profileDir, "profile-installation.yaml"))).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(profileDir, "profile-installation.yaml"), []byte(":!not yaml:/!"), 0755)).To(Succeed())
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile installation:")))
		})
	})

	When("the new profile cannot be found", func() {
		BeforeEach(func() {
			fakeCatalogManager.ShowReturnsOnCall(0, profilesv1.ProfileCatalogEntry{}, fmt.Errorf("whoops"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(fakeCatalogManager.ShowCallCount()).To(Equal(1))
			Expect(err).To(MatchError(ContainSubstring("failed to get profile \"my-profile\" in catalog \"my-catalog\" version \"v0.1.1\":")))
			_, catalogName, profileName, desiredVersion := fakeCatalogManager.ShowArgsForCall(0)
			Expect(catalogName).To(Equal("my-catalog"))
			Expect(profileName).To(Equal("my-profile"))
			Expect(desiredVersion).To(Equal("v0.1.1"))
		})
	})

	When("creating the repo fails", func() {
		BeforeEach(func() {
			fakeRepoManager.CreateRepoWithContentReturns(fmt.Errorf("foo"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(fakeRepoManager.CreateRepoWithContentCallCount()).To(Equal(1))
			Expect(err).To(MatchError("failed to create repository for upgrade: foo"))
		})
	})

	When("creating the user-changes branch fails", func() {
		BeforeEach(func() {
			fakeRepoManager.CreateBranchWithContentFromMainReturns(fmt.Errorf("bar"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(fakeRepoManager.CreateBranchWithContentFromMainCallCount()).To(Equal(1))
			Expect(err).To(MatchError("failed to create branch with user changes: bar"))
		})
	})

	When("creating the user-changes branch fails", func() {
		BeforeEach(func() {
			fakeRepoManager.CreateBranchWithContentFromMainReturnsOnCall(1, fmt.Errorf("baz"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(fakeRepoManager.CreateBranchWithContentFromMainCallCount()).To(Equal(2))
			Expect(err).To(MatchError("failed to create branch with update changes: baz"))
		})
	})

	When("merging branches fails", func() {
		BeforeEach(func() {
			fakeRepoManager.MergeBranchesReturns(nil, fmt.Errorf("bab"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(fakeRepoManager.MergeBranchesCallCount()).To(Equal(1))
			Expect(err).To(MatchError("failed to merge updates with user changes: bab"))
		})
	})
})
