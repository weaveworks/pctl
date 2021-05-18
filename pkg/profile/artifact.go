package profile

import (
	"fmt"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// MakeArtifacts generates artifacts without owners for manual applying to
// a personal cluster.
func MakeArtifacts(sub profilesv1.ProfileSubscription) ([]runtime.Object, error) {
	def, err := getProfileDefinition(sub.Spec.ProfileURL, sub.Spec.Version, strings.Split(sub.Spec.Version, "/")[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	p := newProfile(def, sub)
	return p.makeArtifacts([]string{p.profileRepo()})
}

func (p *Profile) profileRepo() string {
	if p.subscription.Spec.Version != "" {
		return p.subscription.Spec.ProfileURL + ":" + p.subscription.Spec.Version
	}
	return p.subscription.Spec.ProfileURL + ":" + p.subscription.Spec.Branch + ":" + p.subscription.Spec.Path
}

func (p *Profile) makeArtifacts(profileRepos []string) ([]runtime.Object, error) {
	var (
		objs   []runtime.Object
		gitRes *sourcev1.GitRepository
	)
	profileRepoPath := GetProfilePathFromSpec(p.subscription.Spec)

	for _, artifact := range p.definition.Spec.Artifacts {
		if err := artifact.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed for artifact %s: %w", artifact.Name, err)
		}
		switch artifact.Kind {
		case profilesv1.ProfileKind:
			branchOrTag := artifact.Profile.Branch
			path := artifact.Profile.Path
			if artifact.Profile.Version != "" {
				branchOrTag = artifact.Profile.Version
				path = strings.Split(artifact.Profile.Version, "/")[0]
			}
			nestedProfileDef, err := getProfileDefinition(artifact.Profile.URL, branchOrTag, path)
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
			nestedObjs, err := nestedSub.makeArtifacts(profileRepos)
			if err != nil {
				return nil, fmt.Errorf("failed to generate resources for nested profile %q: %w", artifact.Name, err)
			}
			objs = append(objs, nestedObjs...)
		case profilesv1.HelmChartKind:
			objs = append(objs, p.makeHelmRelease(artifact, profileRepoPath))
			if artifact.Path != "" && gitRes == nil {
				// this resource is added at the end because it's generated once.
				gitRes = p.makeGitRepository()
			}
			if artifact.Chart != nil {
				objs = append(objs, p.makeHelmRepository(artifact.Chart.URL, artifact.Chart.Name))
			}
		case profilesv1.KustomizeKind:
			objs = append(objs, p.makeKustomization(artifact, profileRepoPath))
			if gitRes == nil {
				// this resource is added at the end because it's generated once.
				gitRes = p.makeGitRepository()
			}
		default:
			return nil, fmt.Errorf("artifact kind %q not recognized", artifact.Kind)
		}
	}

	// Add the git res as the first object to be created.
	if gitRes != nil {
		objs = append([]runtime.Object{gitRes}, objs...)
	}
	return objs, nil
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
