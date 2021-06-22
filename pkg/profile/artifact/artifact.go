package artifact

import "k8s.io/apimachinery/pkg/runtime"

// Artifact contains the name and objects belonging to a profile artifact
type Artifact struct {
	Objects      []runtime.Object
	Name         string
	RepoURL      string
	PathsToCopy  []string
	SparseFolder string
	Branch       string
}
