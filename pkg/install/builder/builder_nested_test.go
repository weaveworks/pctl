package builder_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("Builder", func() {
	Context("when the artifact is a nested artifact", func() {
		BeforeEach(func() {
			kustomizeFilesDir := filepath.Join(gitDir, "weaveworks-nginx", "files")
			Expect(os.MkdirAll(kustomizeFilesDir, 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(kustomizeFilesDir, "file1"), []byte("foo"), 0755)).To(Succeed())
			artifacts = []artifact.Artifact{
				{
					Artifact: profilesv1.Artifact{
						Name: artifactName,
						Kustomize: &profilesv1.Kustomize{
							Path: "files/",
						},
					},
					ProfileRepoKey: repoKey,
					ProfilePath:    profilePath,
					NestedDirName:  "nested-profile",
				},
			}
		})

		It("places the artifactr in a subdirectory", func() {
			err := artifactBuilder.Write(installation, artifacts, repoLocationMap)
			Expect(err).NotTo(HaveOccurred())

			var files []string
			err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					files = append(files, strings.TrimPrefix(path, rootDir+"/"))
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ConsistOf(
				"artifacts/nested-profile/1/kustomization.yaml",
				"artifacts/nested-profile/1/kustomize-flux.yaml",
				"artifacts/nested-profile/1/files/file1",
				"profile-installation.yaml",
			))
		})
	})
})
