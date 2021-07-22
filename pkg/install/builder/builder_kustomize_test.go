package builder_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"
)

var kustomizeTypeMeta = metav1.TypeMeta{
	Kind:       "Kustomization",
	APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
}

var _ = Describe("Kustomize", func() {
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
				NestedDirName:  "",
			},
		}
	})

	It("generates the Kustomization and copies the raw artifacts into the directory", func() {
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
			"artifacts/1/kustomization.yaml",
			"artifacts/1/kustomize-flux.yaml",
			"artifacts/1/files/file1",
			"profile-installation.yaml",
		))
		kustomization := types.Kustomization{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/kustomization.yaml"), &kustomization)
		Expect(kustomization).To(Equal(types.Kustomization{
			Resources: []string{"kustomize-flux.yaml"},
		}))

		kustomize := kustomizev1.Kustomization{}
		decodeFile(filepath.Join(rootDir, "artifacts/1/kustomize-flux.yaml"), &kustomize)
		Expect(kustomize).To(Equal(kustomizev1.Kustomization{
			TypeMeta: kustomizeTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-%s", installationName, profilePath, artifactName),
				Namespace: namespace,
			},
			Spec: kustomizev1.KustomizationSpec{
				Path: filepath.Join(rootDir, "artifacts/1/files/"),
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind:      "GitRepository",
					Namespace: gitRepoNamespace,
					Name:      gitRepoName,
				},
				Interval:        metav1.Duration{Duration: 300000000000},
				Prune:           true,
				TargetNamespace: namespace,
			},
		}))

		i := profilesv1.ProfileInstallation{}
		decodeFile(filepath.Join(rootDir, "profile-installation.yaml"), &i)
		Expect(i).To(Equal(installation))
	})
})
