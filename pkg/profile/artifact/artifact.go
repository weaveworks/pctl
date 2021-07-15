package artifact

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/types"
)

// Object is an artifact object which adds path to a runtime object.
// path adds location detail to help generate this object.
type Object struct {
	runtime.Object
	// Path, if set, will be used as folder location for this Object.
	Path string
	// Name, if given, will overwrite what this object's filename representation should be.
	Name string
}

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	// Generated Kubernetes resources for this artifact.
	Objects []Object
	// Kustomize resource which limits the number of objects scanned by flux.
	Kustomize *types.Kustomization
	// KustomizeWrapper limits flux to only pick up a specific resource.
	KustomizeWrapper *types.Kustomization
	Name             string
	RepoURL          string
	PathsToCopy      []string
	SparseFolder     string
	Branch           string
	// If set, use this as folder for all artifacts generated from Objects.
	SubFolder string
}
