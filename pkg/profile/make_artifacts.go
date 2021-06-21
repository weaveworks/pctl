package profile

import (
	"errors"
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
	definition, err := getProfileDefinition(installation.Spec.Source.URL, branchOrTag, path, pam.GitClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	var artifacts []artifact.Artifact

	for _, artifact := range definition.Spec.Artifacts {
		if pam.nestedName != "" {
			artifact.Name = filepath.Join(pam.nestedName, artifact.Name)
		}

		var builder Builder
		if artifact.Profile != nil {
			profileRepoName := profileRepo(installation)
			if containsKey(pam.profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", artifact.Profile.Source.URL, artifact.Profile.Source.Branch)
			}
			pam.profileRepos = append(pam.profileRepos, profileRepoName)
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
			pam.nestedName = artifact.Name
			nestedArtifacts, err := MakeArtifacts(pam, *nestedProfile)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
			pam.nestedName = ""
			pam.profileRepos = nil
			continue
		} else if artifact.Kustomize != nil {
			builder = pam.Builders[KUSTOMIZE]
		} else if artifact.Chart != nil {
			builder = pam.Builders[CHART]
		} else {
			return nil, errors.New("no artifact set")
		}
		arts, err := builder.Build(artifact, installation, definition)
		if err != nil {
			return nil, fmt.Errorf("failed to build artifact: %w", err)
		}
		artifacts = append(artifacts, arts...)
	}
	return artifacts, nil
}
