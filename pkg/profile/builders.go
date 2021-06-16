package profile

import profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

const (
	// PROFILE is a profile builder type
	PROFILE = iota
	// KUSTOMIZE is a kustomization builder type
	KUSTOMIZE
	// CHART is a chart builder type
	CHART
)

// Builder can build an artifacts from an installation and a profile artifact.
type Builder interface {
	// Build a single artifact from a profile artifact and installation.
	Build(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]Artifact, error)
}
