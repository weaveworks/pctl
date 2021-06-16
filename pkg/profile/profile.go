package profile

import (
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/repo"
)

// ProfileGetter is a func that can fetch a profile definition
type ProfileGetter func(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error)

var getProfileDefinition = repo.GetProfileDefinition
