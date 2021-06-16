package profile_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/profile"
)

var _ = Describe("Profile", func() {
	var (
		fakeGitClient *fakegit.FakeGit
	)

	BeforeEach(func() {
		fakeGitClient = &fakegit.FakeGit{}
		fakeGitClient.SparseCloneStub = func(url string, branch string, dir string, p string) error {
			from := filepath.Join("testdata", "simple_with_nested", p)
			err := copy.Copy(from, filepath.Join(dir, p))
			Expect(err).NotTo(HaveOccurred())
			return nil
		}
	})
	When("there is a single profile repository", func() {
		It("creates files for all artifacts", func() {
			pSub := profilesv1.ProfileInstallation{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      installationName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL:    "https://github.com/weaveworks/nginx-profile",
						Branch: "main",
					},
				},
			}
			profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
				return profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nginx",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Profile",
						APIVersion: "packages.weave.works.io/profilesv1",
					},
					Spec: profilesv1.ProfileDefinitionSpec{
						ProfileDescription: profilesv1.ProfileDescription{
							Name:        "nginx",
							Description: "foo",
						},
						Artifacts: []profilesv1.Artifact{
							{
								Name: "bitnami-nginx",
								Chart: &profilesv1.Chart{
									URL:     "https://charts.bitnami.com/bitnami",
									Name:    "nginx",
									Version: "8.9.1",
								},
							},
						},
					},
				}, nil
			})
			tempDir, err := ioutil.TempDir("", "catalog-install")
			Expect(err).NotTo(HaveOccurred())
			maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				ProfileName:      "generate-test",
				GitClient:        fakeGitClient,
				RootDir:          tempDir,
				GitRepoNamespace: gitRepoNamespace,
				GitRepoName:      gitRepoName,
			})
			err = maker.Make(pSub)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			profileFile := filepath.Join(tempDir, "generate-test", "profile-installation.yaml")
			artifactHelmRelease := filepath.Join(tempDir, "generate-test", "artifacts", "bitnami-nginx", "HelmRelease.yaml")
			artifactHelmRepository := filepath.Join(tempDir, "generate-test", "artifacts", "bitnami-nginx", "HelmRepository.yaml")
			Expect(files).To(ConsistOf(artifactHelmRepository, artifactHelmRelease, profileFile))

			Expect(hasCorrectFilePerms(profileFile)).To(BeTrue())
			Expect(hasCorrectFilePerms(artifactHelmRelease)).To(BeTrue())
			Expect(hasCorrectFilePerms(artifactHelmRepository)).To(BeTrue())

			content, err := ioutil.ReadFile(profileFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: mySub
  namespace: default
spec:
  source:
    branch: main
    url: https://github.com/weaveworks/nginx-profile
status: {}
`))

			content, err = ioutil.ReadFile(artifactHelmRelease)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(`apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  creationTimestamp: null
  name: mySub-nginx-bitnami-nginx
  namespace: default
spec:
  chart:
    spec:
      chart: nginx
      sourceRef:
        kind: HelmRepository
        name: mySub-nginx-profile-nginx
        namespace: default
      version: 8.9.1
  interval: 0s
status: {}
`))
		})
	})
})
