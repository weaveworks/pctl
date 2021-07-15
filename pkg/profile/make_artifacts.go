package profile

import (
	"fmt"
	"path/filepath"
	"strings"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/profile/artifact"
)

// MakeArtifactsFunc defines a method to create artifacts from an installation using a maker.
type MakeArtifactsFunc func(pam *ProfilesArtifactsMaker, installation profilesv1.ProfileInstallation) ([]artifact.Artifact, error)

var profilesArtifactsMaker = MakeArtifacts

// MakeArtifacts makes profile artifacts.
func MakeArtifacts(pam *ProfilesArtifactsMaker, installation profilesv1.ProfileInstallation) ([]artifact.Artifact, error) {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	definition, err := pam.GetProfileDefinition(installation.Spec.Source.URL, branchOrTag, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	var artifacts []artifact.Artifact

	for _, a := range definition.Spec.Artifacts {
		if pam.nestedName != "" {
			a.Name = filepath.Join(pam.nestedName, a.Name)
		}
		if a.Profile != nil {
			profileRepoName := profileRepo(installation)
			if containsKey(pam.profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", a.Profile.Source.URL, a.Profile.Source.Branch)
			}
			pam.profileRepos = append(pam.profileRepos, profileRepoName)
			nestedProfile := installation.DeepCopyObject().(*profilesv1.ProfileInstallation)
			nestedProfile.Spec.Source.URL = a.Profile.Source.URL
			nestedProfile.Spec.Source.Branch = a.Profile.Source.Branch
			nestedProfile.Spec.Source.Tag = a.Profile.Source.Tag
			nestedProfile.Spec.Source.Path = a.Profile.Source.Path
			if a.Profile.Source.Tag != "" {
				path := "."
				splitTag := strings.Split(a.Profile.Source.Tag, "/")
				if len(splitTag) > 1 {
					path = splitTag[0]
				}
				nestedProfile.Spec.Source.Path = path
			}
			pam.nestedName = a.Name
			nestedArtifacts, err := MakeArtifacts(pam, *nestedProfile)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
			pam.nestedName = ""
			pam.profileRepos = nil
			continue
		}
		arts, err := pam.Builder.Build(a, installation, definition)
		if err != nil {
			return nil, fmt.Errorf("failed to build artifact: %w", err)
		}
		artifacts = append(artifacts, arts...)
	}
	return artifacts, nil
}
