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
	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/upgrade"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Upgrade", func() {
	var (
		fakeCatalogClient  *fakes.FakeCatalogClient
		fakeCatalogManager *fakes.FakeCatalogManager
		fakeGitClient      *fakegit.FakeGit
		cfg                upgrade.UpgradeConfig
		path               string
		version            string
		tempDir            string
		workingDir         string
	)

	BeforeEach(func() {
		var err error
		fakeCatalogClient = new(fakes.FakeCatalogClient)
		fakeCatalogManager = new(fakes.FakeCatalogManager)
		fakeGitClient = new(fakegit.FakeGit)
		version = "v0.1.1"
		tempDir, err = ioutil.TempDir("", "upgrade-test")
		Expect(err).NotTo(HaveOccurred())
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())
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
			},
		}
		bytes, err := yaml.Marshal(installation)
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(filepath.Join(tempDir, "profile-installation.yaml"), bytes, 0755)).To(Succeed())
		path = tempDir
		fakeGitClient.GetDirectoryReturns(workingDir)

		// httpBody := []byte(`{"item":
		// {
		// "name": "my-profile",
		// "description": "nginx 1",
		// "tag": "0.1.0",
		// "catalogSource": "my-catalog (https://github.com/weaveworks/profiles)",
		// "url": "https://github.com/weaveworks/nginx-profile",
		// "prerequisites": ["Kubernetes 1.18+"],
		// "maintainer": "WeaveWorks <gitops@weave.works>"
		// }}
		//   `)
		// fakeCatalogClient.DoRequestReturnsOnCall(0, httpBody, 200, nil)
		fakeCatalogManager.ShowReturnsOnCall(0, profilesv1.ProfileCatalogEntry{
			Tag:           "v0.1.0",
			CatalogSource: "my-catalog",
			Name:          "my-profile2",
		}, nil)

		// httpBody = []byte(`{"item":
		// {
		// "name": "my-profile",
		// "description": "nginx 1",
		// "tag": "0.1.1",
		// "catalogSource": "my-catalog (https://github.com/weaveworks/profiles)",
		// "url": "https://github.com/weaveworks/nginx-profile",
		// "prerequisites": ["Kubernetes 1.18+"],
		// "maintainer": "WeaveWorks <gitops@weave.works>"
		// }}
		//   `)
		// fakeCatalogClient.DoRequestReturnsOnCall(1, httpBody, 200, nil)
		cfg = upgrade.UpgradeConfig{
			ProfileDir:       path,
			Version:          version,
			CatalogClient:    fakeCatalogClient,
			CatalogManager:   fakeCatalogManager,
			GitClient:        fakeGitClient,
			GitRepoName:      "foo",
			GitRepoNamespace: "bar",
		}
	})

	AfterEach(func() {
		// fmt.Println("working dir ", workingDir)
		// fmt.Println("temp dir ", tempDir)
		_ = os.RemoveAll(tempDir)
	})

	It("Upgrades the profile installation", func() {
		err := upgrade.Upgrade(cfg)
		Expect(err).NotTo(HaveOccurred())
		_, catalogName, profileName, desiredVersion := fakeCatalogManager.ShowArgsForCall(0)
		Expect(catalogName).To(Equal("my-catalog"))
		Expect(profileName).To(Equal("my-profile"))
		Expect(desiredVersion).To(Equal("v0.1.0"))

		_, catalogName, profileName, desiredVersion = fakeCatalogManager.ShowArgsForCall(1)
		Expect(catalogName).To(Equal("my-catalog"))
		Expect(profileName).To(Equal("my-profile"))
		Expect(desiredVersion).To(Equal("v0.1.1"))

		Expect(fakeGitClient.InitCallCount()).To(Equal(1))
		Expect(fakeCatalogManager.InstallCallCount()).To(Equal(2))
		Expect(fakeCatalogManager.InstallArgsForCall(0).ProfileConfig).To(Equal(catalog.ProfileConfig{
			CatalogName: "my-catalog",
			ProfileName: "my-profile",
			Version:     "v0.1.0",
			ConfigMap:   "my-config-map",
		}))

		Expect(fakeCatalogManager.InstallArgsForCall(1).ProfileConfig).To(Equal(catalog.ProfileConfig{
			CatalogName: "my-catalog",
			ProfileName: "my-profile",
			Version:     "v0.1.1",
			ConfigMap:   "my-config-map",
		}))
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
			Expect(os.Remove(filepath.Join(tempDir, "profile-installation.yaml"))).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(tempDir, "profile-installation.yaml"), []byte(":!not yaml:/!"), 0755)).To(Succeed())
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile installation:")))
		})
	})

	When("the existing profile cannot be found", func() {
		BeforeEach(func() {
			fakeCatalogManager.ShowReturnsOnCall(0, profilesv1.ProfileCatalogEntry{}, fmt.Errorf("whoops"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to get profile \"my-profile\" in catalog \"my-catalog\" version \"v0.1.0\":")))
			_, catalogName, profileName, desiredVersion := fakeCatalogManager.ShowArgsForCall(0)
			Expect(catalogName).To(Equal("my-catalog"))
			Expect(profileName).To(Equal("my-profile"))
			Expect(desiredVersion).To(Equal("v0.1.0"))
		})
	})

	When("the desired profile cannot be found", func() {
		BeforeEach(func() {
			fakeCatalogManager.ShowReturnsOnCall(1, profilesv1.ProfileCatalogEntry{}, fmt.Errorf("whoops"))
		})

		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to get profile \"my-profile\" in catalog \"my-catalog\" version \"v0.1.1\":")))
			_, catalogName, profileName, desiredVersion := fakeCatalogManager.ShowArgsForCall(1)
			Expect(catalogName).To(Equal("my-catalog"))
			Expect(profileName).To(Equal("my-profile"))
			Expect(desiredVersion).To(Equal("v0.1.1"))
		})
	})

	When("creating the git repository fails", func() {
		BeforeEach(func() {
			fakeGitClient.InitReturns(fmt.Errorf("foo"))
		})
		It("returns an error", func() {
			err := upgrade.Upgrade(cfg)
			Expect(err).To(MatchError(ContainSubstring("failed to init git repo")))
		})
	})
})
