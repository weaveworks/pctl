package artifact

import (
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
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

// Kustomize contains resources which direct flux to what to install.
type Kustomize struct {
	// LocalResourceLimiter resource which limits the number of objects scanned by flux.
	LocalResourceLimiter *types.Kustomization
	// ObjectWrapper is the top most Kubernetes object wrapper for all resources.
	ObjectWrapper *types.Kustomization
}

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	// Generated Kubernetes resources for this artifact.
	Objects      []Object
	Kustomize    Kustomize
	Name         string
	RepoURL      string
	PathsToCopy  []string
	SparseFolder string
	Branch       string
	SubFolder    string
}

type Artifact2 struct {
	profilesv1.Artifact
	NestedProfileDir string
	ProfileName      string
	RepoKey          string
}
