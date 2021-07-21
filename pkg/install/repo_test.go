package install_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/install"
)

var _ = Describe("Repo", func() {
	var (
		fakeGitClient   *fakegit.FakeGit
		repoURL, branch string
	)

	BeforeEach(func() {
		fakeGitClient = &fakegit.FakeGit{}
		repoURL = "github.com/foo/bar"
		branch = "main"
	})

	It("returns the profile definition", func() {
		profileYaml := []byte(`
apiVersion: packages.weave.works.io/v1alpha1
kind: Profile
metadata:
  name: nginx
spec:
  description: foo
  artifacts:
    - name: bar
      kustomize:
        path: baz`)

		path := "my-profile"
		fakeGitClient.CloneStub = func(url string, branch string, dir string) error {
			err := os.MkdirAll(filepath.Join(dir, path), 0755)
			Expect(err).NotTo(HaveOccurred())
			return ioutil.WriteFile(filepath.Join(dir, path, "profile.yaml"), profileYaml, 0755)
		}
		installer := install.NewInstaller(install.Config{
			GitClient:        fakeGitClient,
			RootDir:          "root-dir",
			GitRepoNamespace: "git-repo-namespace",
			GitRepoName:      "git-repo-name",
		})
		definition, err := installer.GetProfileDefinition(repoURL, branch, path)
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeGitClient.CloneCallCount()).To(Equal(1))
		// dir is semi random
		url, cloneBranch, path := fakeGitClient.CloneArgsForCall(0)
		Expect(url).To(Equal(repoURL))
		Expect(cloneBranch).To(Equal(branch))
		Expect(path).To(Equal(path))
		Expect(definition).To(Equal(profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Profile",
				APIVersion: "packages.weave.works.io/v1alpha1",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				ProfileDescription: profilesv1.ProfileDescription{
					Description: "foo",
				},
				Artifacts: []profilesv1.Artifact{
					{
						Name: "bar",
						Kustomize: &profilesv1.Kustomize{
							Path: "baz",
						},
					},
				},
			},
		}))
	})

	When("the clone request fails", func() {
		It("returns an error", func() {
			fakeGitClient.CloneReturns(errors.New("errored"))
			installer := install.NewInstaller(install.Config{
				GitClient:        fakeGitClient,
				RootDir:          "root-dir",
				GitRepoNamespace: "git-repo-namespace",
				GitRepoName:      "git-repo-name",
			})
			_, err := installer.GetProfileDefinition(repoURL, branch, "path")
			Expect(err).To(MatchError("failed to clone the repo: errored"))
			Expect(fakeGitClient.CloneCallCount()).To(Equal(1))
			// dir is semi random
			url, cloneBranch, _ := fakeGitClient.CloneArgsForCall(0)
			Expect(url).To(Equal(repoURL))
			Expect(cloneBranch).To(Equal(branch))
		})
	})
	When("the profile.yaml is not valid yaml", func() {
		It("returns an error", func() {
			profileYaml := []byte("{not valid yaml}")
			path := "my-profile"
			fakeGitClient.CloneStub = func(url string, branch string, dir string) error {
				err := os.MkdirAll(filepath.Join(dir, path), 0755)
				Expect(err).NotTo(HaveOccurred())
				return ioutil.WriteFile(filepath.Join(dir, path, "profile.yaml"), profileYaml, 0755)
			}
			installer := install.NewInstaller(install.Config{
				GitClient:        fakeGitClient,
				RootDir:          "root-dir",
				GitRepoNamespace: "git-repo-namespace",
				GitRepoName:      "git-repo-name",
			})
			_, err := installer.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile")))
			url, cloneBranch, _ := fakeGitClient.CloneArgsForCall(0)
			Expect(url).To(Equal(repoURL))
			Expect(cloneBranch).To(Equal(branch))
		})
	})
	When("the profile.yaml empty", func() {
		It("returns an error", func() {
			profileYaml := []byte("")
			path := "my-profile"
			fakeGitClient.CloneStub = func(url string, branch string, dir string) error {
				err := os.MkdirAll(filepath.Join(dir, path), 0755)
				Expect(err).NotTo(HaveOccurred())
				return ioutil.WriteFile(filepath.Join(dir, path, "profile.yaml"), profileYaml, 0755)
			}
			installer := install.NewInstaller(install.Config{
				GitClient:        fakeGitClient,
				RootDir:          "root-dir",
				GitRepoNamespace: "git-repo-namespace",
				GitRepoName:      "git-repo-name",
			})
			_, err := installer.GetProfileDefinition(repoURL, branch, "my-profile")
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile")))
			url, cloneBranch, _ := fakeGitClient.CloneArgsForCall(0)
			Expect(url).To(Equal(repoURL))
			Expect(cloneBranch).To(Equal(branch))
		})
	})
	Context("caching", func() {
		var (
			profileYaml []byte
			path        string
			installer   *install.Installer
		)
		BeforeEach(func() {
			profileYaml = []byte(`
apiVersion: packages.weave.works.io/v1alpha1
kind: Profile
metadata:
  name: nginx
spec:
  description: foo
  artifacts:
    - name: bar
      kustomize:
        path: baz`)
			path = "my-profile"
			fakeGitClient.CloneStub = func(url string, branch string, dir string) error {
				err := os.MkdirAll(filepath.Join(dir, path), 0755)
				Expect(err).NotTo(HaveOccurred())
				return ioutil.WriteFile(filepath.Join(dir, path, "profile.yaml"), profileYaml, 0755)
			}
			installer = install.NewInstaller(install.Config{
				GitClient:        fakeGitClient,
				RootDir:          "root-dir",
				GitRepoNamespace: "git-repo-namespace",
				GitRepoName:      "git-repo-name",
			})
		})
		When("called multiple times with the same repo", func() {
			It("it only clones it once", func() {
				_, err := installer.GetProfileDefinition(repoURL, branch, "my-profile")
				Expect(err).NotTo(HaveOccurred())
				_, err = installer.GetProfileDefinition(repoURL, branch, "my-profile")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeGitClient.CloneCallCount()).To(Equal(1))
			})
		})
		When("called multiple times with different url", func() {
			It("it clones all", func() {
				_, err := installer.GetProfileDefinition(repoURL, branch, "my-profile")
				Expect(err).NotTo(HaveOccurred())
				_, err = installer.GetProfileDefinition("https://github.com/different", branch, "my-profile")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeGitClient.CloneCallCount()).To(Equal(2))
			})
		})
		When("called multiple times with the same url but different branch", func() {
			It("it clones all", func() {
				_, err := installer.GetProfileDefinition(repoURL, branch, "my-profile")
				Expect(err).NotTo(HaveOccurred())
				_, err = installer.GetProfileDefinition(repoURL, "different", "my-profile")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeGitClient.CloneCallCount()).To(Equal(2))
			})
		})
	})
})
