package artifact

import (
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/types"
)

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects                  []runtime.Object
	Kustomize                *types.Kustomization
	HelmWrapper              *types.Kustomization
	HelmWrapperKustomization *kustomizev1.Kustomization
	Name                     string
	RepoURL                  string
	PathsToCopy              []string
	SparseFolder             string
	Branch                   string
	// If set, use this as folder for all artifacts generated from Objects.
	SubFolder string
}
