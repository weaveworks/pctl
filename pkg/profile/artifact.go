package profile

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
)

//Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects []runtime.Object
	Name    string
}

// MakeArtifacts generates artifacts without owners for manual applying to
// a personal cluster.
func MakeArtifacts(sub profilesv1.ProfileSubscription, gitClient git.Git) ([]Artifact, error) {
	version := sub.Spec.Version
	path := strings.Split(sub.Spec.Version, "/")[0]
	if sub.Spec.Version == "" {
		version = sub.Spec.Branch
		path = sub.Spec.Path
	}
	def, err := getProfileDefinition(sub.Spec.ProfileURL, version, path, gitClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	p := newProfile(def, sub)
	return p.makeArtifacts([]string{p.profileRepo()}, gitClient)
}

func (p *Profile) profileRepo() string {
	if p.subscription.Spec.Version != "" {
		return p.subscription.Spec.ProfileURL + ":" + p.subscription.Spec.Version
	}
	return p.subscription.Spec.ProfileURL + ":" + p.subscription.Spec.Branch + ":" + p.subscription.Spec.Path
}

func (p *Profile) makeArtifacts(profileRepos []string, gitClient git.Git) ([]Artifact, error) {
	var artifacts []Artifact
	profileRepoPath := GetProfilePathFromSpec(p.subscription.Spec)

	for _, artifact := range p.definition.Spec.Artifacts {
		if err := artifact.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed for artifact %s: %w", artifact.Name, err)
		}

		a := Artifact{Name: artifact.Name}

		switch artifact.Kind {
		case profilesv1.ProfileKind:
			branchOrTag := artifact.Profile.Branch
			path := artifact.Profile.Path
			if artifact.Profile.Version != "" {
				branchOrTag = artifact.Profile.Version
				path = strings.Split(artifact.Profile.Version, "/")[0]
			}
			nestedProfileDef, err := getProfileDefinition(artifact.Profile.URL, branchOrTag, path, gitClient)
			if err != nil {
				return nil, fmt.Errorf("failed to get profile definition %s on branch %s: %w", artifact.Profile.URL, branchOrTag, err)
			}
			nestedProfile := p.subscription.DeepCopyObject().(*profilesv1.ProfileSubscription)
			nestedProfile.Spec.ProfileURL = artifact.Profile.URL
			nestedProfile.Spec.Branch = artifact.Profile.Branch
			nestedProfile.Spec.Version = artifact.Profile.Version
			nestedProfile.Spec.Path = artifact.Profile.Path

			nestedSub := newProfile(nestedProfileDef, *nestedProfile)
			profileRepoName := nestedSub.profileRepo()
			if containsKey(profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", artifact.Profile.URL, artifact.Profile.Branch)
			}
			profileRepos = append(profileRepos, profileRepoName)
			nestedArtifacts, err := nestedSub.makeArtifacts(profileRepos, gitClient)
			if err != nil {
				return nil, fmt.Errorf("failed to generate resources for nested profile %q: %w", artifact.Name, err)
			}
			for i := range nestedArtifacts {
				nestedArtifacts[i].Name = filepath.Join(artifact.Name, nestedArtifacts[i].Name)
			}
			artifacts = append(artifacts, nestedArtifacts...)
		case profilesv1.HelmChartKind:
			a.Objects = append(a.Objects, p.makeHelmRelease(artifact, profileRepoPath))
			if artifact.Path != "" {
				a.Objects = append(a.Objects, p.makeGitRepository())
			}
			if artifact.Chart != nil {
				a.Objects = append(a.Objects, p.makeHelmRepository(artifact.Chart.URL, artifact.Chart.Name))
			}
			artifacts = append(artifacts, a)
		case profilesv1.KustomizeKind:
			a.Objects = append(a.Objects, p.makeKustomization(artifact, profileRepoPath))
			a.Objects = append(a.Objects, p.makeGitRepository())
			artifacts = append(artifacts, a)
		default:
			return nil, fmt.Errorf("artifact kind %q not recognized", artifact.Kind)
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
	return join(p.subscription.Name, p.definition.Name, name)
}

// GetProfilePathFromSpec returns either the path to the profile in the repo. Extracted from the
// version field or directly from path
func GetProfilePathFromSpec(spec profilesv1.ProfileSubscriptionSpec) string {
	if spec.Path != "" {
		return spec.Path
	}
	return strings.Split(spec.Version, "/")[0]
}

func join(s ...string) string {
	return strings.Join(s, "-")
}
