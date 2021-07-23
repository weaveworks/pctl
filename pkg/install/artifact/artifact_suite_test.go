package artifact_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestArtifact(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artiact Suite")
}

var (
	rootDir          string
	gitDir           string
	artifactWriter   *artifact.Writer
	gitRepoName      = "my-git-repo"
	gitRepoNamespace = "my-git-repo-namespace"
	profileURL       = "github.com/weaveworks/profiles-examples"
	profileBranch    = "main"
	profilePath      = "path/to/profile"
	profileName      = "weaveworks-nginx"
	installation     profilesv1.ProfileInstallation
	installationName = "install-name"
	namespace        = "my-namespace"
	artifacts        []artifact.ArtifactWrapper
	artifactName     = "1"
)

var _ = BeforeEach(func() {
	var err error
	rootDir, err = ioutil.TempDir("", "root-dir")
	Expect(err).NotTo(HaveOccurred())
	gitDir, err = ioutil.TempDir("", "root-dir")
	Expect(err).NotTo(HaveOccurred())

	installation = profilesv1.ProfileInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installationName,
			Namespace: namespace,
		},
		Spec: profilesv1.ProfileInstallationSpec{
			Source: &profilesv1.Source{
				URL:    profileURL,
				Branch: profileBranch,
				Path:   profilePath,
			},
		},
	}

	artifactWriter = &artifact.Writer{
		GitRepositoryName:      gitRepoName,
		GitRepositoryNamespace: gitRepoNamespace,
		RootDir:                rootDir,
	}

})

var _ = AfterEach(func() {
	_ = os.RemoveAll(rootDir)
	_ = os.RemoveAll(gitDir)
})

func decodeFile(filepath string, obj interface{}) {
	content, err := ioutil.ReadFile(filepath)
	Expect(err).NotTo(HaveOccurred())

	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(obj)
	Expect(err).NotTo(HaveOccurred())
}
