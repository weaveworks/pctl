package profile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/otiai10/copy"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/profile/artifact"
	"github.com/weaveworks/pctl/pkg/profile/chart"
	"github.com/weaveworks/pctl/pkg/profile/kustomize"
)

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
	builders := map[int]Builder{
		KUSTOMIZE: &kustomize.Builder{
			Config: kustomize.Config{
				GitRepositoryName:      cfg.GitRepoName,
				GitRepositoryNamespace: cfg.GitRepoNamespace,
				RootDir:                cfg.RootDir,
			},
		},
		CHART: &chart.Builder{
			Config: chart.Config{
				GitRepositoryName:      cfg.GitRepoName,
				GitRepositoryNamespace: cfg.GitRepoNamespace,
				RootDir:                cfg.RootDir,
			},
		},
	}
	return &ProfilesArtifactsMaker{
		MakerConfig: cfg,
		Builders:    builders,
	}
}

// Make generates artifacts without owners for manual applying to a personal cluster.
func (pa *ProfilesArtifactsMaker) Make(installation profilesv1.ProfileInstallation) error {
	artifacts, err := profilesArtifactsMaker(pa, installation)
	if err != nil {
		return fmt.Errorf("failed to build artifact: %w", err)
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
		if artifact.Kustomize != nil {
			data, err := yaml.Marshal(artifact.Kustomize)
			if err != nil {
				return err
			}
			filename := filepath.Join(artifactDir, "kustomization.yaml")
			err = os.WriteFile(filename, data, 0644)
			if err != nil {
				return err
			}
		}
	}

	return pa.generateOutput(filepath.Join(profileRootdir, "profile-installation.yaml"), &installation)
}

// getRepositoryLocalArtifacts clones all repository local artifacts so they can be copied over to the flux repository.
func (pa *ProfilesArtifactsMaker) getRepositoryLocalArtifacts(a artifact.Artifact, artifactDir string) error {
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

func profileRepo(installation profilesv1.ProfileInstallation) string {
	if installation.Spec.Source.Tag != "" {
		return installation.Spec.Source.URL + ":" + installation.Spec.Source.Tag
	}
	return installation.Spec.Source.URL + ":" + installation.Spec.Source.Branch + ":" + installation.Spec.Source.Path
}

func containsKey(list []string, key string) bool {
	for _, value := range list {
		if value == key {
			return true
		}
	}
	return false
}
