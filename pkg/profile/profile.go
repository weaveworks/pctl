package profile

import (
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/repo"
)

// Profile contains information and interfaces required for creating and
// managing profile artefacts (child resources)
type Profile struct {
	definition             profilesv1.ProfileDefinition
	subscription           profilesv1.ProfileSubscription
	nestedName             string
	rootDir                string
	gitRepositoryName      string
	gitRepositoryNamespace string
}

// ProfileGetter is a func that can fetch a profile definition
type ProfileGetter func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error)

var getProfileDefinition = repo.GetProfileDefinition

// New returns a new Profile object
func newProfile(def profilesv1.ProfileDefinition, sub profilesv1.ProfileSubscription, rootDir, gitRepoNamespace, gitRepoName string) *Profile {
	return &Profile{
		definition:             def,
		subscription:           sub,
		gitRepositoryName:      gitRepoName,
		gitRepositoryNamespace: gitRepoNamespace,
		rootDir:                rootDir,
	}
}
