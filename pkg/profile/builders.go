package profile

import profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

const (
	// KUSTOMIZE is a kustomization builder type
	KUSTOMIZE = iota
	// CHART is a chart builder type
	CHART
)

// BuilderConfig defines some common configuration values for builders.
type BuilderConfig struct {
	GitRepositoryName      string
	GitRepositoryNamespace string
	RootDir                string
}

// Builder can build an artifacts from an installation and a profile artifact.
//go:generate counterfeiter -o fakes/builder_maker.go . Builder
type Builder interface {
	// Build a single artifact from a profile artifact and installation.
	Build(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]Artifact, error)
}
