package profile_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	fakegit "github.com/weaveworks/pctl/pkg/git/fakes"
	"github.com/weaveworks/pctl/pkg/profile"
)

const (
	subscriptionName     = "mySub"
	namespace            = "default"
	branch               = "main"
	profileName1         = "profileName"
	profileName2         = "profileName2"
	chartName1           = "chartOneArtifactName"
	chartPath1           = "chart/artifact/path-one"
	chartName2           = "chartTwoArtifactName"
	chartPath2           = "chart/artifact/path-two"
	helmChartName1       = "helmChartArtifactName1"
	helmChartChart1      = "helmChartChartName1"
	helmChartURL1        = "https://org.github.io/chart"
	helmChartVersion1    = "8.8.1"
	kustomizeName1       = "kustomizeOneArtifactName"
	kustomizePath1       = "kustomize/artifact/path-one"
	profileSubKind       = "ProfileInstallation"
	profileSubAPIVersion = "weave.works/v1alpha1"
	profileURL           = "https://github.com/org/repo-name"
	gitRepoNamespace     = "git-repo-namespace"
	gitRepoName          = "git-repo-name"
	rootDir              = "root-dir"
)

var (
	profileTypeMeta = metav1.TypeMeta{
		Kind:       profileSubKind,
		APIVersion: profileSubAPIVersion,
	}

	gitRepoKind  = sourcev1.GitRepositoryKind
	helmRepoKind = sourcev1.HelmRepositoryKind
)

var _ = Describe("Profile", func() {
	var (
		p             *profile.Profile
		pSub          profilesv1.ProfileInstallation
		pDef          profilesv1.ProfileDefinition
		pNestedDef    profilesv1.ProfileDefinition
		pNestedDefURL = "https://github.com/org/repo-name-nested"
		fakeGitClient *fakegit.FakeGit
	)

	BeforeEach(func() {
		pSub = profilesv1.ProfileInstallation{
			TypeMeta: profileTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      subscriptionName,
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
							Path: chartPath1,
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
								URL:    pNestedDefURL,
								Branch: "main",
								Path:   profileName2,
							},
						},
					},
					{
						Name: chartName2,
						Chart: &profilesv1.Chart{
							Path: chartPath2,
						},
					},
					{
						Name: kustomizeName1,
						Kustomize: &profilesv1.Kustomize{
							Path: kustomizePath1,
						},
					},
					{
						Name: helmChartName1,
						Chart: &profilesv1.Chart{
							URL:     helmChartURL1,
							Name:    helmChartChart1,
							Version: helmChartVersion1,
						},
					},
				},
			},
		}
		fakeGitClient = &fakegit.FakeGit{}
		p.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
			if profileURL == repoURL {
				return pDef, nil
			}
			return pNestedDef, nil
		})
	})

	Context("MakeArtifacts", func() {
		It("generates the artifacts", func() {
			maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
				GitClient:        fakeGitClient,
				RootDir:          rootDir,
				GitRepoNamespace: gitRepoNamespace,
				GitRepoName:      gitRepoName,
			})
			artifacts, err := maker.MakeArtifacts(pSub)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifacts).To(HaveLen(4))

			By("generating the nested profile artifact", func() {
				nestedProfileArtifact := artifacts[0]
				Expect(nestedProfileArtifact.Name).To(Equal(filepath.Join(profileName2, chartName1)))

				objects := nestedProfileArtifact.Objects
				Expect(objects).To(HaveLen(1))

				helmReleaseName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName2, chartName1)
				helmRelease := objects[0].(*helmv2.HelmRelease)

				Expect(helmRelease.Name).To(Equal(helmReleaseName))
				Expect(helmRelease.Spec.Chart.Spec.Chart).To(Equal("root-dir/artifacts/profileName2/chartOneArtifactName/chart/artifact/path-one"))
				Expect(helmRelease.Spec.Chart.Spec.SourceRef).To(Equal(
					helmv2.CrossNamespaceObjectReference{
						Kind:      gitRepoKind,
						Name:      gitRepoName,
						Namespace: gitRepoNamespace,
					},
				))
				Expect(helmRelease.GetValues()).To(Equal(map[string]interface{}{
					"replicaCount": float64(3),
					"service": map[string]interface{}{
						"port": float64(8081),
					},
				}))
				Expect(helmRelease.Spec.ValuesFrom).To(Equal([]helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
					},
				}))
			})

			By("generating the path based helm release artifact", func() {
				pathBasedHelmArtifact := artifacts[1]
				Expect(pathBasedHelmArtifact.Name).To(Equal(chartName2))

				objects := pathBasedHelmArtifact.Objects

				helmReleaseName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName1, chartName2)
				helmRelease := objects[0].(*helmv2.HelmRelease)
				Expect(helmRelease.Name).To(Equal(helmReleaseName))
				Expect(err).NotTo(HaveOccurred())
				Expect(helmRelease.Spec.Chart.Spec.Chart).To(Equal("root-dir/artifacts/chartTwoArtifactName/chart/artifact/path-two"))
				Expect(helmRelease.Spec.Chart.Spec.SourceRef).To(Equal(
					helmv2.CrossNamespaceObjectReference{
						Kind:      gitRepoKind,
						Name:      gitRepoName,
						Namespace: gitRepoNamespace,
					},
				))
				Expect(helmRelease.GetValues()).To(Equal(map[string]interface{}{
					"replicaCount": float64(3),
					"service": map[string]interface{}{
						"port": float64(8081),
					},
				}))
				Expect(helmRelease.Spec.ValuesFrom).To(Equal([]helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
					},
				}))
			})

			By("generating the kustomize artifact", func() {
				kustomizeArtifact := artifacts[2]
				Expect(kustomizeArtifact.Name).To(Equal(kustomizeName1))

				objects := kustomizeArtifact.Objects

				kustomizeName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName1, kustomizeName1)
				kustomize := objects[0].(*kustomizev1.Kustomization)
				Expect(kustomize.Name).To(Equal(kustomizeName))
				Expect(kustomize.Spec.Path).To(Equal("root-dir/artifacts/kustomizeOneArtifactName/kustomize/artifact/path-one"))
				Expect(kustomize.Spec.TargetNamespace).To(Equal(namespace))
				Expect(kustomize.Spec.Prune).To(BeTrue())
				Expect(kustomize.Spec.Interval).To(Equal(metav1.Duration{Duration: time.Minute * 5}))
				Expect(kustomize.Spec.SourceRef).To(Equal(
					kustomizev1.CrossNamespaceSourceReference{
						Kind:      gitRepoKind,
						Name:      gitRepoName,
						Namespace: gitRepoNamespace,
					},
				))
			})

			By("generating the repository based helm artifact", func() {
				helmArtifact := artifacts[3]
				Expect(helmArtifact.Name).To(Equal(helmChartName1))

				objects := helmArtifact.Objects
				Expect(objects).To(HaveLen(2))

				helmRefName := fmt.Sprintf("%s-%s-%s", subscriptionName, "repo-name", helmChartChart1)
				helmRepo := objects[1].(*sourcev1.HelmRepository)
				Expect(helmRepo.Name).To(Equal(helmRefName))
				Expect(helmRepo.Spec.URL).To(Equal(helmChartURL1))

				helmReleaseName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName1, helmChartName1)
				helmRelease := objects[0].(*helmv2.HelmRelease)
				Expect(helmRelease.Name).To(Equal(helmReleaseName))
				Expect(helmRelease.Spec.Chart.Spec.Chart).To(Equal(helmChartChart1))
				Expect(helmRelease.Spec.Chart.Spec.Version).To(Equal(helmChartVersion1))
				Expect(helmRelease.Spec.Chart.Spec.SourceRef).To(Equal(
					helmv2.CrossNamespaceObjectReference{
						Kind:      helmRepoKind,
						Name:      helmRefName,
						Namespace: namespace,
					},
				))
				Expect(helmRelease.GetValues()).To(Equal(map[string]interface{}{
					"replicaCount": float64(3),
					"service": map[string]interface{}{
						"port": float64(8081),
					},
				}))
				Expect(helmRelease.Spec.ValuesFrom).To(Equal([]helmv2.ValuesReference{
					{
						Name:     "nginx-values",
						Kind:     "Secret",
						Optional: true,
					},
				}))
			})
		})

		When("the branch name for a git repository is not domain compatible", func() {
			It("will sanitise it", func() {
				pSub = profilesv1.ProfileInstallation{
					TypeMeta: profileTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						Name:      subscriptionName,
						Namespace: namespace,
					},
					Spec: profilesv1.ProfileInstallationSpec{
						Source: &profilesv1.Source{
							URL:    profileURL,
							Branch: "not_domain_compatible",
						},
					},
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGitClient,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				artifacts, err := maker.MakeArtifacts(pSub)
				Expect(err).NotTo(HaveOccurred())
				Expect(artifacts).To(HaveLen(4))

				By("generating the path based helm release artifact", func() {
					pathBasedHelmArtifact := artifacts[1]
					Expect(pathBasedHelmArtifact.Name).To(Equal(chartName2))

					objects := pathBasedHelmArtifact.Objects

					helmReleaseName := fmt.Sprintf("%s-%s-%s", subscriptionName, profileName1, chartName2)
					helmRelease := objects[0].(*helmv2.HelmRelease)
					Expect(helmRelease.Name).To(Equal(helmReleaseName))
					Expect(err).NotTo(HaveOccurred())
					Expect(helmRelease.Spec.Chart.Spec.Chart).To(Equal("root-dir/artifacts/chartTwoArtifactName/chart/artifact/path-two"))
					Expect(helmRelease.Spec.Chart.Spec.SourceRef).To(Equal(
						helmv2.CrossNamespaceObjectReference{
							Kind:      gitRepoKind,
							Name:      gitRepoName,
							Namespace: gitRepoNamespace,
						},
					))
				})
			})
		})

		When("the git repository name is not defined", func() {
			It("errors out", func() {
				pSub = profilesv1.ProfileInstallation{
					TypeMeta: profileTypeMeta,
					ObjectMeta: metav1.ObjectMeta{
						Name:      subscriptionName,
						Namespace: namespace,
					},
					Spec: profilesv1.ProfileInstallationSpec{
						Source: &profilesv1.Source{
							URL:    profileURL,
							Branch: branch,
						},
					},
				}
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient: fakeGitClient,
					RootDir:   rootDir,
				})
				artifacts, err := maker.MakeArtifacts(pSub)
				Expect(err).To(MatchError("failed to generate resources for nested profile \"profileName2\": in case of local resources, the flux gitrepository object's details must be provided"))
				Expect(artifacts).To(BeEmpty())
			})
		})

		When("fetching the nested profile definition fails", func() {
			BeforeEach(func() {
				p.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
					if repoURL == profileURL {
						return pDef, nil
					}
					return pNestedDef, fmt.Errorf("foo")
				})
			})

			It("returns an error", func() {
				maker := profile.NewProfilesArtifactsMaker(profile.MakerConfig{
					GitClient:        fakeGitClient,
					RootDir:          rootDir,
					GitRepoNamespace: gitRepoNamespace,
					GitRepoName:      gitRepoName,
				})
				_, err := maker.MakeArtifacts(pSub)
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to get profile definition %s on branch %s: foo", pNestedDefURL, branch))))
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
					_, err := maker.MakeArtifacts(pSub)
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
					_, err := maker.MakeArtifacts(pSub)
					Expect(err).To(MatchError(ContainSubstring("failed to generate resources for nested profile \"profileName2\":")))
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
					_, err := maker.MakeArtifacts(pSub)
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
					_, err := maker.MakeArtifacts(pSub)
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
					_, err := maker.MakeArtifacts(pSub)
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
					_, err := maker.MakeArtifacts(pSub)
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

					p.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
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
					_, err := maker.MakeArtifacts(pSub)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", pNestedDefURL, branch))))
				})
			})
		})
	})
	Context("GenerateArtifactsOutput", func() {
		It("creates files for all artifacts", func() {
			pSub = profilesv1.ProfileInstallation{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      subscriptionName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL:    "https://github.com/weaveworks/profiles-examples",
						Branch: "main",
						Path:   "weaveworks-nginx",
					},
				},
			}
			p.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
				return profilesv1.ProfileDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "weaveworks-nginx",
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
								Name: "nginx-deployment",
								Kustomize: &profilesv1.Kustomize{
									Path: "nginx/deployment",
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
			artifacts, err := maker.MakeArtifacts(pSub)
			Expect(err).NotTo(HaveOccurred())
			fakeGitClient.SparseCloneStub = func(url string, branch string, dir string, p string) error {
				fullPath := filepath.Join(dir, "weaveworks-nginx", "nginx", "deployment")
				err := os.MkdirAll(fullPath, 0755)
				Expect(err).NotTo(HaveOccurred())
				return ioutil.WriteFile(filepath.Join(fullPath, "deployment.yaml"), []byte("validYaml:"), 0644)
			}
			err = maker.GenerateArtifactsOutput(artifacts, pSub)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			profileFile := filepath.Join(tempDir, "generate-test", "profile.yaml")
			artifactFile := filepath.Join(tempDir, "generate-test", "artifacts", "nginx-deployment", "Kustomization.yaml")
			artifactFileDeployment := filepath.Join(tempDir, "generate-test", "artifacts", "nginx-deployment", "nginx", "deployment", "deployment.yaml")
			Expect(files).To(ConsistOf(artifactFile, artifactFileDeployment, profileFile))

			Expect(hasCorrectFilePerms(profileFile)).To(BeTrue())
			Expect(hasCorrectFilePerms(artifactFile)).To(BeTrue())
			Expect(hasCorrectFilePerms(artifactFileDeployment)).To(BeTrue())

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
    path: weaveworks-nginx
    url: https://github.com/weaveworks/profiles-examples
status: {}
`))

			content, err = ioutil.ReadFile(artifactFile)
			Expect(err).NotTo(HaveOccurred())
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
`, tempDir)))
		})
	})

	When("there is a flat profile repository", func() {
		It("creates files for all artifacts", func() {
			pSub = profilesv1.ProfileInstallation{
				TypeMeta: profileTypeMeta,
				ObjectMeta: metav1.ObjectMeta{
					Name:      subscriptionName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL:    "https://github.com/weaveworks/nginx-profile",
						Branch: "main",
					},
				},
			}
			p.SetProfileGetter(func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
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
			artifacts, err := maker.MakeArtifacts(pSub)
			Expect(err).NotTo(HaveOccurred())
			err = maker.GenerateArtifactsOutput(artifacts, pSub)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			profileFile := filepath.Join(tempDir, "generate-test", "profile.yaml")
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
