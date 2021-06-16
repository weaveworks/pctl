package profile

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/otiai10/copy"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"knative.dev/pkg/apis"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
)

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects      []runtime.Object
	Name         string
	RepoURL      string
	PathsToCopy  []string
	SparseFolder string
	Branch       string
}

// ArtifactsMaker can create a list of artifacts.
//go:generate counterfeiter -o fakes/artifacts_maker.go . ArtifactsMaker
type ArtifactsMaker interface {
	Make(installation profilesv1.ProfileInstallation) error
}

// MakerConfig contains all configuration properties for the Artifacts Maker.
type MakerConfig struct {
	GitClient        git.Git
	RootDir          string
	GitRepoNamespace string
	GitRepoName      string
	ProfileName      string
}

// ProfilesArtifactsMaker creates a list of artifacts from profiles data.
type ProfilesArtifactsMaker struct {
	MakerConfig

	Builders     map[int]Builder
	nestedName   string
	profileRepos []string
}

// NewProfilesArtifactsMaker creates a new profiles artifacts maker.
func NewProfilesArtifactsMaker(cfg MakerConfig) *ProfilesArtifactsMaker {
	builderConfig := BuilderConfig{
		GitRepositoryName:      cfg.GitRepoName,
		GitRepositoryNamespace: cfg.GitRepoNamespace,
		RootDir:                cfg.RootDir,
	}
	builders := map[int]Builder{
		KUSTOMIZE: &KustomizeBuilder{
			BuilderConfig: builderConfig,
		},
		CHART: &ChartBuilder{
			BuilderConfig: builderConfig,
		},
	}
	return &ProfilesArtifactsMaker{
		MakerConfig: cfg,
		Builders:    builders,
	}
}

// Make generates artifacts without owners for manual applying to a personal cluster.
func (pa *ProfilesArtifactsMaker) Make(installation profilesv1.ProfileInstallation) error {
	artifacts, err := pa.makeArtifacts(installation)
	if err != nil {
		return err
	}
	profileRootdir := filepath.Join(pa.RootDir, pa.ProfileName)
	artifactsRootDir := filepath.Join(profileRootdir, "artifacts")
	for _, artifact := range artifacts {
		artifactDir := filepath.Join(artifactsRootDir, artifact.Name)
		if err := os.MkdirAll(artifactDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory")
		}
		if artifact.RepoURL != "" {
			if err := pa.getRepositoryLocalArtifacts(artifact, artifactDir); err != nil {
				return fmt.Errorf("failed to get package local artifacts: %w", err)
			}
		}
		for _, obj := range artifact.Objects {
			filename := filepath.Join(artifactDir, fmt.Sprintf("%s.%s", obj.GetObjectKind().GroupVersionKind().Kind, "yaml"))
			if err := pa.generateOutput(filename, obj); err != nil {
				return err
			}
		}
	}

	return pa.generateOutput(filepath.Join(profileRootdir, "profile-installation.yaml"), &installation)
}

// makeArtifacts creates artifacts. This is a separate function from the main Make function in order to handle
// nested profiles recursively.
func (pa *ProfilesArtifactsMaker) makeArtifacts(installation profilesv1.ProfileInstallation) ([]Artifact, error) {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	def, err := getProfileDefinition(installation.Spec.Source.URL, branchOrTag, path, pa.GitClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	var artifacts []Artifact

	for _, artifact := range def.Spec.Artifacts {
		if pa.nestedName != "" {
			artifact.Name = filepath.Join(pa.nestedName, artifact.Name)
		}

		if err := validateArtifact(artifact); err != nil {
			return nil, fmt.Errorf("validation failed for artifact %s: %w", artifact.Name, err)
		}

		t := -1
		if artifact.Profile != nil {
			profileRepoName := profileRepo(installation)
			if containsKey(pa.profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", artifact.Profile.Source.URL, artifact.Profile.Source.Branch)
			}
			pa.profileRepos = append(pa.profileRepos, profileRepoName)
			nestedArtifacts, err := pa.handleNestedProfile(artifact, installation)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
			pa.nestedName = ""
			pa.profileRepos = nil
			continue
		} else if artifact.Kustomize != nil {
			t = KUSTOMIZE
		} else if artifact.Chart != nil {
			t = CHART
		}
		if t == -1 {
			return nil, errors.New("no artifact set")
		}
		arts, err := pa.Builders[t].Build(artifact, installation, def)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, arts...)
	}
	return artifacts, nil
}

// getRepositoryLocalArtifacts clones all repository local artifacts so they can be copied over to the flux repository.
func (pa *ProfilesArtifactsMaker) getRepositoryLocalArtifacts(a Artifact, artifactDir string) error {
	u := uuid.NewString()[:6]
	tmp, err := ioutil.TempDir("", "sparse_clone_git_repo_"+u)
	if err != nil {
		return fmt.Errorf("failed to create temp folder: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			fmt.Println("Failed to remove tmp folder: ", tmp)
		}
	}()
	profilePath := a.SparseFolder
	branch := a.Branch
	if err := pa.GitClient.SparseClone(a.RepoURL, branch, tmp, profilePath); err != nil {
		return fmt.Errorf("failed to sparse clone folder with url: %s; branch: %s; path: %s; with error: %w",
			a.RepoURL,
			branch,
			profilePath,
			err)
	}
	for _, path := range a.PathsToCopy {
		// nginx/chart/...
		if strings.Contains(path, string(os.PathSeparator)) {
			path = filepath.Dir(path)
		}
		fullPath := filepath.Join(tmp, profilePath, path)
		if err := copy.Copy(fullPath, filepath.Join(artifactDir, path)); err != nil {
			return fmt.Errorf("failed to move folder: %w", err)
		}
	}
	return nil
}

func (pa *ProfilesArtifactsMaker) generateOutput(filename string, o runtime.Object) error {
	e := kjson.NewSerializerWithOptions(kjson.DefaultMetaFactory, nil, nil, kjson.SerializerOptions{Yaml: true, Strict: true})
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to properly close file %s\n", f.Name())
		}
	}(f)
	if err := e.Encode(o, f); err != nil {
		return err
	}
	return nil

}

// handleNestedProfile takes care of creating nested profile configuration.
func (pa *ProfilesArtifactsMaker) handleNestedProfile(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation) ([]Artifact, error) {
	nestedProfile := installation.DeepCopyObject().(*profilesv1.ProfileInstallation)
	nestedProfile.Spec.Source.URL = artifact.Profile.Source.URL
	nestedProfile.Spec.Source.Branch = artifact.Profile.Source.Branch
	nestedProfile.Spec.Source.Tag = artifact.Profile.Source.Tag
	nestedProfile.Spec.Source.Path = artifact.Profile.Source.Path
	if artifact.Profile.Source.Tag != "" {
		path := "."
		splitTag := strings.Split(artifact.Profile.Source.Tag, "/")
		if len(splitTag) > 1 {
			path = splitTag[0]
		}
		nestedProfile.Spec.Source.Path = path
	}
	pa.nestedName = artifact.Name
	return pa.makeArtifacts(*nestedProfile)
}

func profileRepo(installation profilesv1.ProfileInstallation) string {
	if installation.Spec.Source.Tag != "" {
		return installation.Spec.Source.URL + ":" + installation.Spec.Source.Tag
	}
	return installation.Spec.Source.URL + ":" + installation.Spec.Source.Branch + ":" + installation.Spec.Source.Path
}

func makeArtifactName(name string, installationName, definitionName string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return join(installationName, definitionName, name)
}

func containsKey(list []string, key string) bool {
	for _, value := range list {
		if value == key {
			return true
		}
	}
	return false
}

func join(s ...string) string {
	return strings.Join(s, "-")
}

func validateArtifact(in profilesv1.Artifact) error {
	if in.Chart != nil && in.Profile != nil {
		return apis.ErrMultipleOneOf("chart", "profile")
	}

	if in.Profile != nil && in.Kustomize != nil {
		return apis.ErrMultipleOneOf("profile", "kustomize")
	}

	if in.Chart != nil && in.Kustomize != nil {
		return apis.ErrMultipleOneOf("chart", "kustomize")
	}

	if in.Chart != nil && in.Chart.Path != "" && in.Chart.URL != "" {
		return apis.ErrMultipleOneOf("chart.path", "chart.url")
	}

	return nil
}
