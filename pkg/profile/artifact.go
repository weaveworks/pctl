package profile

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
)

//Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects      []runtime.Object
	Name         string
	RepoURL      string
	PathsToCopy  []string
	SparseFolder string
	Branch       string
}

// MakeArtifacts generates artifacts without owners for manual applying to
// a personal cluster.
func MakeArtifacts(installation profilesv1.ProfileInstallation, gitClient git.Git, rootDir, gitRepoNamespace, gitRepoName string) ([]Artifact, error) {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	def, err := getProfileDefinition(installation.Spec.Source.URL, branchOrTag, path, gitClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	p := newProfile(def, installation, rootDir, gitRepoNamespace, gitRepoName)
	return p.makeArtifacts([]string{p.profileRepo()}, gitClient)
}

func (p *Profile) profileRepo() string {
	if p.subscription.Spec.Source.Tag != "" {
		return p.subscription.Spec.Source.URL + ":" + p.subscription.Spec.Source.Tag
	}
	return p.subscription.Spec.Source.URL + ":" + p.subscription.Spec.Source.Branch + ":" + p.subscription.Spec.Source.Path
}

func (p *Profile) makeArtifacts(profileRepos []string, gitClient git.Git) ([]Artifact, error) {
	var artifacts []Artifact
	profileRepoPath := p.subscription.Spec.Source.Path

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
			nestedProfile := p.subscription.DeepCopyObject().(*profilesv1.ProfileInstallation)
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
				branch := p.subscription.Spec.Source.Branch
				if p.subscription.Spec.Source.Tag != "" {
					branch = p.subscription.Spec.Source.Tag
				}
				a.RepoURL = p.subscription.Spec.Source.URL
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
			branch := p.subscription.Spec.Source.Branch
			if p.subscription.Spec.Source.Tag != "" {
				branch = p.subscription.Spec.Source.Tag
			}
			a.RepoURL = p.subscription.Spec.Source.URL
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
	return join(p.subscription.Name, p.definition.Name, name)
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
