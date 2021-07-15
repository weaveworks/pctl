package shared

import (
	"path/filepath"
	"strings"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder is a builder which contains functionality that is shared between all builders.
type Builder struct {
}

// ContainsArtifact checks whether an artifact with a specific name exists in a list of artifacts.
func (k *Builder) ContainsArtifact(name string, stack []profilesv1.Artifact) (profilesv1.Artifact, bool) {
	for _, a := range stack {
		if a.Name == name {
			return a, true
		}
	}
	return profilesv1.Artifact{}, false
}

// MakeArtifactName creates a name for an artifact.
func (k Builder) MakeArtifactName(name string, installationName, definitionName string) string {
	// if this is a nested artifact, it's name contains a /
	if strings.Contains(name, "/") {
		name = filepath.Base(name)
	}
	return k.Join(installationName, definitionName, name)
}

// Join creates a joined string of name using - as a join character.
func (k Builder) Join(s ...string) string {
	return strings.Join(s, "-")
}

// MakeKustomization creates a Kustomize object.
func (k Builder) MakeKustomization(
	artifact profilesv1.Artifact,
	repoPath string,
	installation profilesv1.ProfileInstallation,
	definitionName string,
	dependencies []profilesv1.Artifact,
	gitRepoName, gitRepoNamespace string,
) *kustomizev1.Kustomization {
	var dependsOn []dependency.CrossNamespaceDependencyReference
	for _, dep := range dependencies {
		dependsOn = append(dependsOn, dependency.CrossNamespaceDependencyReference{
			Name:      k.MakeArtifactName(dep.Name, installation.Name, definitionName),
			Namespace: installation.Namespace,
		})
	}
	return &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.MakeArtifactName(artifact.Name, installation.Name, definitionName),
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
				Name:      gitRepoName,
				Namespace: gitRepoNamespace,
			},
			DependsOn: dependsOn,
		},
	}
}
