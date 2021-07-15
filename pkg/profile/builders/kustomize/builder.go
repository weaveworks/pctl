package kustomize

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"

	"github.com/weaveworks/pctl/pkg/profile/artifact"
)

// Config defines some common configuration values for builders.
type Config struct {
	GitRepositoryName      string
	GitRepositoryNamespace string
	RootDir                string
}

// Builder will build kustomize resources.
type Builder struct {
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
		d, ok := containsArtifact(dep.Name, definition.Spec.Artifacts)
		if !ok {
			return nil, fmt.Errorf("%s's depending artifact %s not found in the list of artifacts", a.Name, dep.Name)
		}
		deps = append(deps, d)
	}

	a.Objects = append(a.Objects, k.makeKustomization(att, path, installation, definition.Name, deps))
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

func (k *Builder) makeKustomization(artifact profilesv1.Artifact, repoPath string, installation profilesv1.ProfileInstallation, definitionName string, dependencies []profilesv1.Artifact) *kustomizev1.Kustomization {
	var dependsOn []dependency.CrossNamespaceDependencyReference
	for _, dep := range dependencies {
		dependsOn = append(dependsOn, dependency.CrossNamespaceDependencyReference{
			Name:      makeArtifactName(dep.Name, installation.Name, definitionName),
			Namespace: installation.Namespace,
		})
	}
	return &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeArtifactName(artifact.Name, installation.Name, definitionName),
			Namespace: installation.ObjectMeta.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       kustomizev1.KustomizationKind,
			APIVersion: kustomizev1.GroupVersion.String(),
		},
		Spec: kustomizev1.KustomizationSpec{
			Path:            repoPath,
			Interval:        metav1.Duration{Duration: time.Minute * 5},
			Prune:           true,
			TargetNamespace: installation.ObjectMeta.Namespace,
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind:      sourcev1.GitRepositoryKind,
				Name:      k.GitRepositoryName,
				Namespace: k.GitRepositoryNamespace,
			},
			DependsOn: dependsOn,
		},
	}
}

// containsArtifact checks whether an artifact with a specific name exists in a list of artifacts.
func containsArtifact(name string, stack []profilesv1.Artifact) (profilesv1.Artifact, bool) {
	for _, a := range stack {
		if a.Name == name {
			return a, true
		}
	}
	return profilesv1.Artifact{}, false
}

func makeArtifactName(name string, installationName, definitionName string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return join(installationName, definitionName, name)
}

func join(s ...string) string {
	return strings.Join(s, "-")
}
