package profile_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/otiai10/copy"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/profile"
)

const (
	branch               = "main"
	chartName1           = "nginx-server"
	gitRepoName          = "git-repo-name"
	gitRepoNamespace     = "git-repo-namespace"
	helmChartChart1      = "helmChartChartName1"
	helmChartName1       = "helmChartArtifactName1"
	helmChartURL1        = "https://org.github.io/chart"
	helmChartVersion1    = "8.8.1"
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

var _ = Describe("Profile", func() {
	var (
		pSub          profilesv1.ProfileInstallation
		pDef          profilesv1.ProfileDefinition
		pNestedDef    profilesv1.ProfileDefinition
		pNestedDefURL = "https://github.com/org/repo-name-nested"
		fakeGitClient *fakegit.FakeGit
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
				Values: &apiextensionsv1.JSON{
					Raw: []byte(`{"replicaCount": 3,"service":{"port":8081}}`),
				},
				ValuesFrom: []helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
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
						Name: chartName1,
						Chart: &profilesv1.Chart{
							Path: "nginx/chart",
						},
					},
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
		fakeGitClient = &fakegit.FakeGit{}
		profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
			if path == "weaveworks-nginx" {
				return pDef, nil
			}
			return pNestedDef, nil
		})
		fakeGitClient.SparseCloneStub = func(url string, branch string, dir string, p string) error {
			from := filepath.Join("testdata", "simple_with_nested", p)
			err := copy.Copy(from, filepath.Join(dir, p))
			Expect(err).NotTo(HaveOccurred())
			return nil
		}
		var err error
		rootDir, err = ioutil.TempDir("", "test_make_artifacts")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Make", func() {
		It("generates the artifacts", func() {
			maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				GitClient:        fakeGitClient,
				RootDir:          rootDir,
				GitRepoNamespace: gitRepoNamespace,
				GitRepoName:      gitRepoName,
			})
			err := maker.Make(pSub)
			Expect(err).NotTo(HaveOccurred())
			files := make(map[string]string)
			err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files[fmt.Sprintf("%s/%s", filepath.Base(filepath.Dir(path)), filepath.Base(path))] = path
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			consistsOf := []string{
				filepath.Join(rootDir, "profile-installation.yaml"),
				filepath.Join(rootDir, "artifacts", "nginx-deployment", "Kustomization.yaml"),
				filepath.Join(rootDir, "artifacts", "nginx-deployment", "nginx", "deployment", "deployment.yaml"),
				filepath.Join(rootDir, "artifacts", "bitnami-nginx", "nginx-server", "HelmRelease.yaml"),
				filepath.Join(rootDir, "artifacts", "bitnami-nginx", "nginx-server", "nginx", "chart", "Chart.yaml"),
				filepath.Join(rootDir, "artifacts", "dokuwiki", "HelmRelease.yaml"),
				filepath.Join(rootDir, "artifacts", "dokuwiki", "HelmRepository.yaml"),
			}
			Expect(files).To(ConsistOf(consistsOf))

			By("inspecting the profile-installation.yaml", func() {
				parent := filepath.Base(rootDir)
				content, err := ioutil.ReadFile(files[filepath.Join(parent, "profile-installation.yaml")])
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: weave.works/v1alpha1
kind: ProfileInstallation
metadata:
  creationTimestamp: null
  name: mySub
  namespace: default
spec:
  source:
    branch: main
    path: weaveworks-nginx
    url: https://github.com/org/repo-name
  values:
    replicaCount: 3
    service:
      port: 8081
  valuesFrom:
  - kind: Secret
    name: nginx-values
    optional: true
status: {}
`))
			})

			By("generating the nested profile artifact", func() {
				content, err := ioutil.ReadFile(files["nginx-server/HelmRelease.yaml"])
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  creationTimestamp: null
  name: mySub-bitnami-nginx-nginx-server
  namespace: default
spec:
  chart:
    spec:
      chart: %s/artifacts/bitnami-nginx/nginx-server/nginx/chart
      sourceRef:
        kind: GitRepository
        name: git-repo-name
        namespace: git-repo-namespace
  interval: 0s
  values:
    replicaCount: 3
    service:
      port: 8081
  valuesFrom:
  - kind: Secret
    name: nginx-values
    optional: true
status: {}
`, rootDir)))

			})

			By("generating the path based kustomization artifact", func() {
				content, err := ioutil.ReadFile(files["nginx-deployment/Kustomization.yaml"])
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(fmt.Sprintf(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  creationTimestamp: null
  name: mySub-weaveworks-nginx-nginx-deployment
  namespace: default
spec:
  interval: 5m0s
  path: %s/artifacts/nginx-deployment/nginx/deployment
  prune: true
  sourceRef:
    kind: GitRepository
    name: git-repo-name
    namespace: git-repo-namespace
  targetNamespace: default
status: {}
`, rootDir)))
			})

			By("generating a remote helm chart", func() {
				content, err := ioutil.ReadFile(files["dokuwiki/HelmRelease.yaml"])
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  creationTimestamp: null
  name: mySub-weaveworks-nginx-dokuwiki
  namespace: default
spec:
  chart:
    spec:
      chart: dokuwiki
      sourceRef:
        kind: HelmRepository
        name: mySub-repo-name-dokuwiki
        namespace: default
      version: 11.1.6
  interval: 0s
  values:
    replicaCount: 3
    service:
      port: 8081
  valuesFrom:
  - kind: Secret
    name: nginx-values
    optional: true
status: {}
`))
				content, err = ioutil.ReadFile(files["dokuwiki/HelmRepository.yaml"])
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(Equal(`apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
  creationTimestamp: null
  name: mySub-repo-name-dokuwiki
  namespace: default
spec:
  interval: 0s
  url: https://charts.bitnami.com/bitnami
status: {}
`))
			})
		})

		When("the git repository name is not defined", func() {
			It("errors out", func() {
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient: fakeGitClient,
					RootDir:   rootDir,
				})
				err := maker.Make(pSub)
				Expect(err).To(MatchError("in case of local resources, the flux gitrepository object's details must be provided"))
			})
		})

		When("fetching the nested profile definition fails", func() {
			It("returns an error", func() {
				profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
					return profilesv1.ProfileDefinition{}, fmt.Errorf("foo")
				})
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGitClient,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				err := maker.Make(pSub)
				Expect(err).To(MatchError(ContainSubstring("failed to get profile definition: foo")))
			})
		})

		When("configured with an invalid artifact", func() {
			When("the Kind of artifact is unknown", func() {
				BeforeEach(func() {
					pDef.Spec.Artifacts[0] = profilesv1.Artifact{}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("no artifact set")))
				})
			})
			When("the nested profile is invalid", func() {
				BeforeEach(func() {
					pNestedDef.Spec.Artifacts[0] = profilesv1.Artifact{}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("no artifact set")))
				})
			})
			When("helmRepository and path", func() {
				BeforeEach(func() {
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
									Name: helmChartName1,
									Chart: &profilesv1.Chart{
										URL:     helmChartURL1,
										Name:    helmChartChart1,
										Version: helmChartVersion1,
										Path:    "https://not.empty",
									},
								},
							},
						},
					}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("validation failed for artifact helmChartArtifactName1: expected exactly one, got both: chart.path, chart.url")))
				})
			})
			When("chart and kustomize", func() {
				BeforeEach(func() {
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
									Name: helmChartName1,
									Chart: &profilesv1.Chart{
										URL:     helmChartURL1,
										Name:    helmChartChart1,
										Version: helmChartVersion1,
									},
									Kustomize: &profilesv1.Kustomize{
										Path: "https://not.empty",
									},
								},
							},
						},
					}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("validation failed for artifact helmChartArtifactName1: expected exactly one, got both: chart, kustomize")))
				})
			})
			When("profile and kustomize", func() {
				BeforeEach(func() {
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
									Name: helmChartName1,
									Profile: &profilesv1.Profile{
										Source: &profilesv1.Source{
											URL:    "example.com",
											Branch: "branch",
										},
									},
									Kustomize: &profilesv1.Kustomize{
										Path: "https://not.empty",
									},
								},
							},
						},
					}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("validation failed for artifact helmChartArtifactName1: expected exactly one, got both: kustomize, profile")))
				})
			})

			When("helmRepository and profile", func() {
				BeforeEach(func() {
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
									Name: helmChartName1,
									Chart: &profilesv1.Chart{
										URL:     helmChartURL1,
										Name:    helmChartChart1,
										Version: helmChartVersion1,
									},
									Profile: &profilesv1.Profile{
										Source: &profilesv1.Source{
											URL:    "example.com",
											Branch: "branch",
										},
									},
								},
							},
						},
					}
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("validation failed for artifact helmChartArtifactName1: expected exactly one, got both: chart, profile")))
				})
			})

			When("profile artifact pointing to itself", func() {
				var (
					pNestedDef2    profilesv1.ProfileDefinition
					pNestedDef2URL = "example.com/nested"
				)
				BeforeEach(func() {
					pNestedDef2 = profilesv1.ProfileDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: profileName2,
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
									Name: "recursive",
									Profile: &profilesv1.Profile{
										Source: &profilesv1.Source{
											URL:    profileURL,
											Branch: branch,
										},
									},
								},
							},
						},
					}
					pNestedDef.Spec.Artifacts = []profilesv1.Artifact{
						{
							Name: "recursive",
							Profile: &profilesv1.Profile{
								Source: &profilesv1.Source{
									URL:    pNestedDef2URL,
									Branch: branch,
								},
							},
						},
					}

					profile.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
						if repoURL == profileURL {
							return pDef, nil
						}
						if repoURL == pNestedDef2URL {
							return pNestedDef2, nil
						}
						return pNestedDef, nil
					})
				})

				It("errors", func() {
					maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
						GitClient:        fakeGitClient,
						RootDir:          rootDir,
						GitRepoNamespace: gitRepoNamespace,
						GitRepoName:      gitRepoName,
					})
					err := maker.Make(pSub)
					Expect(err).To(MatchError(ContainSubstring("recursive artifact detected: profile example.com/nested on branch main contains an artifact that points recursively back at itself")))
				})
			})
		})
	})
	When("there is a single profile repository", func() {
		It("creates files for all artifacts", func() {
			pSub = profilesv1.ProfileInstallation{
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

func hasCorrectFilePerms(file string) bool {
	info, err := os.Stat(file)
	Expect(err).NotTo(HaveOccurred())
	return strconv.FormatUint(uint64(info.Mode().Perm()), 8) == strconv.FormatInt(0644, 8)
}
