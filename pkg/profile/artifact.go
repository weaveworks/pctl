package profile

import (
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
}

// NewProfilesArtifactsMaker creates a new profiles artifacts maker.
func NewProfilesArtifactsMaker(cfg MakerConfig) *ProfilesArtifactsMaker {
	return &ProfilesArtifactsMaker{
		MakerConfig: cfg,
	}
}

// Make generates artifacts without owners for manual applying to a personal cluster.
func (pa *ProfilesArtifactsMaker) Make(installation profilesv1.ProfileInstallation) error {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	def, err := getProfileDefinition(installation.Spec.Source.URL, branchOrTag, path, pa.GitClient)
	if err != nil {
		return fmt.Errorf("failed to get profile definition: %w", err)
	}
	p := newProfile(def, installation, pa.RootDir, pa.GitRepoNamespace, pa.GitRepoName)
	artifacts, err := p.makeArtifacts([]string{p.profileRepo()}, pa.GitClient)
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

func (p *Profile) profileRepo() string {
	if p.installation.Spec.Source.Tag != "" {
		return p.installation.Spec.Source.URL + ":" + p.installation.Spec.Source.Tag
	}
	return p.installation.Spec.Source.URL + ":" + p.installation.Spec.Source.Branch + ":" + p.installation.Spec.Source.Path
}

// makeArtifacts will be part of the artifacts maker and not profiles.
func (p *Profile) makeArtifacts(profileRepos []string, gitClient git.Git) ([]Artifact, error) {
	var artifacts []Artifact
	profileRepoPath := p.installation.Spec.Source.Path

	for _, artifact := range p.definition.Spec.Artifacts {
		if err := validateArtifact(artifact); err != nil {
			return nil, fmt.Errorf("validation failed for artifact %s: %w", artifact.Name, err)
		}
		if p.nestedName != "" {
			artifact.Name = filepath.Join(p.nestedName, artifact.Name)
		}
		a := Artifact{Name: artifact.Name}

		if artifact.Profile != nil {
			branchOrTag := artifact.Profile.Source.Branch
			path := artifact.Profile.Source.Path
			if artifact.Profile.Source.Tag != "" {
				branchOrTag = artifact.Profile.Source.Tag
				path = strings.Split(artifact.Profile.Source.Tag, "/")[0]
			}
			nestedProfileDef, err := getProfileDefinition(artifact.Profile.Source.URL, branchOrTag, path, gitClient)
			if err != nil {
				return nil, fmt.Errorf("failed to get profile definition %s on branch %s: %w", artifact.Profile.Source.URL, branchOrTag, err)
			}
			nestedProfile := p.installation.DeepCopyObject().(*profilesv1.ProfileInstallation)
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

			nestedSub := newProfile(nestedProfileDef, *nestedProfile, p.rootDir, p.gitRepositoryNamespace, p.gitRepositoryName)
			nestedSub.nestedName = artifact.Name
			profileRepoName := nestedSub.profileRepo()
			if containsKey(profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", artifact.Profile.Source.URL, artifact.Profile.Source.Branch)
			}
			profileRepos = append(profileRepos, profileRepoName)
			nestedArtifacts, err := nestedSub.makeArtifacts(profileRepos, gitClient)
			if err != nil {
				return nil, fmt.Errorf("failed to generate resources for nested profile %q: %w", artifact.Name, err)
			}
			artifacts = append(artifacts, nestedArtifacts...)
			p.nestedName = ""
		} else if artifact.Chart != nil {
			helmRelease := p.makeHelmRelease(artifact, profileRepoPath)
			a.Objects = append(a.Objects, helmRelease)
			if artifact.Chart.Path != "" {
				if p.gitRepositoryNamespace == "" && p.gitRepositoryName == "" {
					return nil, fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
				}
				helmRelease.Spec.Chart.Spec.Chart = filepath.Join(p.rootDir, "artifacts", artifact.Name, artifact.Chart.Path)
				branch := p.installation.Spec.Source.Branch
				if p.installation.Spec.Source.Tag != "" {
					branch = p.installation.Spec.Source.Tag
				}
				a.RepoURL = p.installation.Spec.Source.URL
				a.SparseFolder = p.definition.Name
				a.Branch = branch
				a.PathsToCopy = append(a.PathsToCopy, artifact.Chart.Path)
			}
			if artifact.Chart.URL != "" {
				helmRepository := p.makeHelmRepository(artifact.Chart.URL, artifact.Chart.Name)
				a.Objects = append(a.Objects, helmRepository)
			}
			artifacts = append(artifacts, a)
		} else if artifact.Kustomize != nil {
			if p.gitRepositoryNamespace == "" && p.gitRepositoryName == "" {
				return nil, fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
			}
			path := filepath.Join(p.rootDir, "artifacts", artifact.Name, artifact.Kustomize.Path)
			a.Objects = append(a.Objects, p.makeKustomization(artifact, path))
			branch := p.installation.Spec.Source.Branch
			if p.installation.Spec.Source.Tag != "" {
				branch = p.installation.Spec.Source.Tag
			}
			a.RepoURL = p.installation.Spec.Source.URL
			a.SparseFolder = p.definition.Name
			a.Branch = branch
			a.PathsToCopy = append(a.PathsToCopy, artifact.Kustomize.Path)
			artifacts = append(artifacts, a)
		} else {
			return nil, fmt.Errorf("no artifact set")
		}
	}
	return artifacts, nil
}

func containsKey(list []string, key string) bool {
	for _, value := range list {
		if value == key {
			return true
		}
	}
	return false
}

func (p *Profile) makeArtifactName(name string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return join(p.installation.Name, p.definition.Name, name)
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
