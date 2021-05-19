package profile

import (
	"github.com/weaveworks/pctl/pkg/repo"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// Profile contains information and interfaces required for creating and
// managing profile artefacts (child resources)
type Profile struct {
	definition   profilesv1.ProfileDefinition
	subscription profilesv1.ProfileSubscription
}

// ProfileGetter is a func that can fetch a profile definition
type ProfileGetter func(repoURL, branch, path string) (profilesv1.ProfileDefinition, error)

var getProfileDefinition = repo.GetProfileDefinition

// New returns a new Profile object
func newProfile(def profilesv1.ProfileDefinition, sub profilesv1.ProfileSubscription) *Profile {
	return &Profile{
		definition:   def,
		subscription: sub,
	}
}
