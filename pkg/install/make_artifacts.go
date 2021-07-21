package install

import (
	"fmt"
	"path/filepath"
	"strings"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/install/artifact"
)

// MakeArtifactsFunc defines a method to create artifacts from an installation using a maker.
type MakeArtifactsFunc func(i *Installer, installation profilesv1.ProfileInstallation) ([]artifact.Artifact, error)

var profilesArtifactsMaker = MakeArtifacts

// MakeArtifacts makes profile artifacts.
func MakeArtifacts(installer *Installer, installation profilesv1.ProfileInstallation) ([]artifact.Artifact, error) {
	path := installation.Spec.Source.Path
	branchOrTag := installation.Spec.Source.Tag
	if installation.Spec.Source.Tag == "" {
		branchOrTag = installation.Spec.Source.Branch
	}
	definition, err := installer.GetProfileDefinition(installation.Spec.Source.URL, branchOrTag, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile definition: %w", err)
	}
	var artifacts []artifact.Artifact

	for _, a := range definition.Spec.Artifacts {
		if installer.nestedName != "" {
			a.Name = filepath.Join(installer.nestedName, a.Name)
		}
		if a.Profile != nil {
			profileRepoName := profileRepo(installation)
			if containsKey(installer.profileRepos, profileRepoName) {
				return nil, fmt.Errorf("recursive artifact detected: profile %s on branch %s contains an artifact that points recursively back at itself", a.Profile.Source.URL, a.Profile.Source.Branch)
			}
			installer.profileRepos = append(installer.profileRepos, profileRepoName)
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
			installer.nestedName = a.Name
			nestedArtifacts, err := MakeArtifacts(installer, *nestedProfile)
			if err != nil {
				return nil, err
			}
			artifacts = append(artifacts, nestedArtifacts...)
			installer.nestedName = ""
			installer.profileRepos = nil
			continue
		}
		arts, err := installer.Builder.Build(a, installation, definition)
		if err != nil {
			return nil, fmt.Errorf("failed to build artifact: %w", err)
		}
		artifacts = append(artifacts, arts...)
	}
	return artifacts, nil
}
