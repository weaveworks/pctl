package artifact

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/types"
)

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects      []runtime.Object
	Kustomize    *types.Kustomization
	Name         string
	RepoURL      string
	PathsToCopy  []string
	SparseFolder string
	Branch       string
}
