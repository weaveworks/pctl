package profile

import (
	"fmt"
	"path/filepath"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KustomizeBuilder will build kustomize resources.
type KustomizeBuilder struct {
	BuilderConfig
}

// Build a single artifact from a profile artifact and installation.
func (k *KustomizeBuilder) Build(artifact profilesv1.Artifact, installation profilesv1.ProfileInstallation, definition profilesv1.ProfileDefinition) ([]Artifact, error) {
	if k.GitRepositoryNamespace == "" && k.GitRepositoryName == "" {
		return nil, fmt.Errorf("in case of local resources, the flux gitrepository object's details must be provided")
	}
	a := Artifact{Name: artifact.Name}
	path := filepath.Join(k.RootDir, "artifacts", artifact.Name, artifact.Kustomize.Path)
	a.Objects = append(a.Objects, k.makeKustomization(artifact, path, installation, definition.Name))
	branch := installation.Spec.Source.Branch
	if installation.Spec.Source.Tag != "" {
		branch = installation.Spec.Source.Tag
	}
	a.RepoURL = installation.Spec.Source.URL
	a.SparseFolder = definition.Name
	a.Branch = branch
	a.PathsToCopy = append(a.PathsToCopy, artifact.Kustomize.Path)
	return []Artifact{a}, nil
}

func (k *KustomizeBuilder) makeKustomization(artifact profilesv1.Artifact, repoPath string, installation profilesv1.ProfileInstallation, definitionName string) *kustomizev1.Kustomization {
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
		},
	}
}
