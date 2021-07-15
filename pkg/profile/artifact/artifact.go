package artifact

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/types"
)

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	// Generated Kubernetes resources for this artifact.
	Objects []runtime.Object
	// Kustomize resource which limits the number of objects scanned by flux.
	Kustomize *types.Kustomization
	// KustomizeWrapper limits flux to only pick up a specific resource.
	KustomizeWrapper *types.Kustomization
	// KustomizeWrapperObject wraps any resource into a Kustomization object.
	KustomizeWrapperObject *kustomizev1.Kustomization
	Name                   string
	RepoURL                string
	PathsToCopy            []string
	SparseFolder           string
	Branch                 string
	// If set, use this as folder for all artifacts generated from Objects.
	SubFolder string
}
