package install_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/install"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	"github.com/weaveworks/pctl/pkg/install/builder/fakes"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var _ = Describe("Installer", func() {
	var (
		fakeGitClient    *fakegit.FakeGit
		fakeBuilder      *fakes.FakeBuilder
		installer        *install.Installer
		installation     profilesv1.ProfileInstallation
		gitRepoName      = "git-repo-name"
		gitRepoNamespace = "git-repo-namespace"
		rootDir          = "/tmp/root/dir"
		installationName = "my-installation"
		namespace        = "my-namespace"

		profileURL3        = "github.com/weaveworks/profiles-examples"
		profileBranch3     = "main"
		profilePath3       = "some-third-path"
		profileDefinition3 = profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "weaveworks-nginx",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				Artifacts: []profilesv1.Artifact{
					{
						Name: "artifact-3",
					},
				},
			},
		}

		profileURL2        = "github.com/weaveworks/nginx-profile"
		profileBranch2     = "main"
		profilePath2       = "nginx-profile"
		profileDefinition2 = profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "weaveworks-nginx",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				Artifacts: []profilesv1.Artifact{
					{
						Name: "artifact-2",
					},
					{
						Name: "nested-artifact-2",
						Profile: &profilesv1.Profile{
							Source: &profilesv1.Source{
								URL:    profileURL3,
								Branch: profileBranch3,
								Path:   profilePath3,
							},
						},
					},
				},
			},
		}

		profileURL1        = "github.com/weaveworks/profiles-examples"
		profileBranch1     = "main"
		profilePath1       = "weaveworks-nginx"
		profileDefinition1 = profilesv1.ProfileDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "weaveworks-nginx",
			},
			Spec: profilesv1.ProfileDefinitionSpec{
				Artifacts: []profilesv1.Artifact{
					{
						Name: "artifact-1",
					},
					{
						Name: "nested-artifact-1",
						Profile: &profilesv1.Profile{
							Source: &profilesv1.Source{
								URL:    profileURL2,
								Branch: profileBranch2,
								Path:   profilePath2,
							},
						},
					},
				},
			},
		}
	)

	BeforeEach(func() {
		fakeGitClient = &fakegit.FakeGit{}
		fakeBuilder = &fakes.FakeBuilder{}
		installer = install.NewInstaller(install.Config{
			GitClient:        fakeGitClient,
			RootDir:          rootDir,
			GitRepoNamespace: gitRepoNamespace,
			GitRepoName:      gitRepoName,
		})
		installer.SetBuilder(fakeBuilder)

		installation = profilesv1.ProfileInstallation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      installationName,
				Namespace: namespace,
			},
			Spec: profilesv1.ProfileInstallationSpec{
				Source: &profilesv1.Source{
					URL:    profileURL1,
					Branch: profileBranch1,
					Path:   profilePath1,
				},
			},
		}
		fakeGitClient.CloneStub = func(url, branch, dir string) error {
			if url == profileURL1 {
				err := writeKubernetesResource(&profileDefinition1, filepath.Join(dir, profilePath1), "profile.yaml")
				if err != nil {
					return err
				}
				return writeKubernetesResource(&profileDefinition3, filepath.Join(dir, profilePath3), "profile.yaml")
			}
			if url == profileURL2 {
				return writeKubernetesResource(&profileDefinition2, filepath.Join(dir, profilePath2), "profile.yaml")
			}
			return fmt.Errorf("this shouldn't of been called")
		}
	})

	It("clones the repositories and invokes build with the artifacts", func() {
		Expect(installer.Install(installation)).To(Succeed())

		By("cloning repositories")
		//profile3 and profile1 share the same repo&branch, so only 2 clones should occur.
		Expect(fakeGitClient.CloneCallCount()).To(Equal(2))
		url, branch, profile1CloneDir := fakeGitClient.CloneArgsForCall(0)
		Expect(url).To(Equal(profileURL1))
		Expect(branch).To(Equal(profileBranch1))
		url, branch, profile2CloneDir := fakeGitClient.CloneArgsForCall(1)
		Expect(url).To(Equal(profileURL2))
		Expect(branch).To(Equal(profileBranch2))

		Expect(fakeBuilder.WriteCallCount()).To(Equal(1))
		inst, artifacts, repoClones := fakeBuilder.WriteArgsForCall(0)
		Expect(inst).To(Equal(installation))

		profile1RepoKey := fmt.Sprintf("%s:%s", profileURL1, profileBranch1)
		profile2RepoKey := fmt.Sprintf("%s:%s", profileURL2, profileBranch2)
		By("creating the artifacts")
		Expect(repoClones).To(Equal(map[string]string{
			profile1RepoKey: profile1CloneDir,
			profile2RepoKey: profile2CloneDir,
		}))
		Expect(artifacts).To(ConsistOf(
			artifact.Artifact{
				Artifact: profilesv1.Artifact{
					Name: "artifact-1",
				},
				ProfilePath: profilePath1,
				ProfileRepoKey:     profile1RepoKey,
			},
			artifact.Artifact{
				Artifact: profilesv1.Artifact{
					Name: "artifact-2",
				},
				NestedDirName: "nested-artifact-1",
				ProfilePath:      profilePath2,
				ProfileRepoKey:          profile2RepoKey,
			},
			artifact.Artifact{
				Artifact: profilesv1.Artifact{
					Name: "artifact-3",
				},
				NestedDirName: "nested-artifact-2",
				ProfilePath:      profilePath3,
				ProfileRepoKey:          profile1RepoKey,
			},
		))
	})

	When("cloning fails", func() {
		It("returns an erorr", func() {
			fakeGitClient.CloneReturns(fmt.Errorf("foo"))
			err := installer.Install(installation)
			Expect(err).To(MatchError(fmt.Sprintf("failed to clone repo %q: foo", profileURL1)))
		})
	})

	When("the profile.yaml doesn't exist", func() {
		It("returns an erorr", func() {

			fakeGitClient.CloneStub = func(url, branch, dir string) error {
				return nil
			}
			err := installer.Install(installation)
			Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to read profile.yaml in repo %q branch %q path %q:", profileURL1, profileBranch1, profilePath1))))
		})
	})

	When("profile.yaml isn't valid", func() {
		It("returns an erorr", func() {
			fakeGitClient.CloneStub = func(url, branch, dir string) error {
				err := os.MkdirAll(filepath.Join(dir, profilePath1), 0755)
				if err != nil {
					return err
				}
				return ioutil.WriteFile(filepath.Join(dir, profilePath1, "profile.yaml"), []byte("!botvalidyaml\n!"), 0755)
			}
			err := installer.Install(installation)
			Expect(err).To(MatchError(ContainSubstring("failed to parse profile.yaml")))
		})
	})
})

func writeKubernetesResource(obj runtime.Object, dir, filename string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to properly close file %s\n", f.Name())
		}
	}(f)
	if err := e.Encode(obj, f); err != nil {
		return err
	}
	return nil
}
