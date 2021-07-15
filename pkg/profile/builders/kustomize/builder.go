package kustomize

import (
	"fmt"
	"path/filepath"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	"knative.dev/pkg/apis"

	"github.com/weaveworks/pctl/pkg/profile/artifact"
	"github.com/weaveworks/pctl/pkg/profile/builders/shared"
)

// Config defines some common configuration values for builders.
type Config struct {
	GitRepositoryName      string
	GitRepositoryNamespace string
	RootDir                string
}

// Builder will build kustomize resources.
type Builder struct {
	shared.Builder
	Config
}

// Build a single artifact from a profile artifact and installation.
func (k *Builder) Build(att profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]artifact.Artifact, error) {
	if k.GitRepositoryNamespace == "" && k.GitRepositoryName == "" {
		return nil, fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
	}
	if err := validateArtifact(att); err != nil {
		return nil, fmt.Errorf("validation failed for artifact %s: %w", att.Name, err)
	}
	a := artifact.Artifact{Name: att.Name}
	path := filepath.Join(k.RootDir, "artifacts", att.Name, att.Kustomize.Path)

	var deps []profilesv1.Artifact
	for _, dep := range att.DependsOn {
		d, ok := k.ContainsArtifact(dep.Name, definition.Spec.Artifacts)
		if !ok {
			return nil, fmt.Errorf("%s's depending artifact %s not found in the list of artifacts", a.Name, dep.Name)
		}
		deps = append(deps, d)
	}

	a.Objects = append(a.Objects, k.MakeKustomization(att, path, installation, definition.Name, deps, k.GitRepositoryName, k.GitRepositoryNamespace))
	branch := installation.Spec.Source.Branch
	if installation.Spec.Source.Tag != "" {
		branch = installation.Spec.Source.Tag
	}
	a.RepoURL = installation.Spec.Source.URL
	a.SparseFolder = definition.Name
	a.Branch = branch
	a.PathsToCopy = append(a.PathsToCopy, att.Kustomize.Path)
	return []artifact.Artifact{a}, nil
}

// validateArtifact validates that the artifact has valid chart properties.
func validateArtifact(in profilesv1.Artifact) error {
	if in.Chart != nil {
		return apis.ErrMultipleOneOf("chart", "kustomize")
	}
	if in.Profile != nil {
		return apis.ErrMultipleOneOf("profile", "kustomize")
	}
	return nil
}
