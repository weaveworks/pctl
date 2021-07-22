package artifact

import (
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

type Artifact struct {
	profilesv1.Artifact
	ParentProfileArtifactName string
	ProfilePath               string
	ProfileRepoKey            string
}
