package builder_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/pkg/install/artifact"
	"github.com/weaveworks/pctl/pkg/install/builder"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Builder Suite")
}

var (
	rootDir          string
	gitDir           string
	artifactBuilder  *builder.ArtifactBuilder2
	repoKey          = "repoKey"
	gitRepoName      = "my-git-repo"
	gitRepoNamespace = "my-git-repo-namespace"
	profileURL       = "github.com/weaveworks/profiles-examples"
	profileBranch    = "main"
	profilePath      = "weaveworks-nginx"
	installation     profilesv1.ProfileInstallation
	installationName = "install-name"
	namespace        = "my-namespace"
	artifacts        []artifact.Artifact
	artifactName     = "1"
	repoLocationMap  map[string]string
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

	artifactBuilder = &builder.ArtifactBuilder2{
		GitRepositoryName:      gitRepoName,
		GitRepositoryNamespace: gitRepoNamespace,
		RootDir:                rootDir,
	}

	repoLocationMap = map[string]string{
		repoKey: gitDir,
	}
})

var _ = AfterEach(func() {
	os.RemoveAll(rootDir)
	os.RemoveAll(gitDir)
})

func decodeFile(filepath string, obj interface{}) {
	content, err := ioutil.ReadFile(filepath)
	Expect(err).NotTo(HaveOccurred())

	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(obj)
	Expect(err).NotTo(HaveOccurred())
}
