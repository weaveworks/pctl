package profile

import (
	"fmt"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Profile) makeGitRepository(path string) *sourcev1.GitRepository {
	ref := &sourcev1.GitRepositoryRef{
		Branch: fmt.Sprintf("%s:%s:%s", p.definition.Name, path, p.subscription.Spec.Branch),
	}
	if p.subscription.Spec.Version != "" {
		ref = &sourcev1.GitRepositoryRef{
			Tag: fmt.Sprintf("%s:%s:%s", p.definition.Name, path, p.subscription.Spec.Version),
		}
	}
	return &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.gitRepositoryName,
			Namespace: p.gitRepositoryNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       sourcev1.GitRepositoryKind,
			APIVersion: sourcev1.GroupVersion.String(),
		},
		Spec: sourcev1.GitRepositorySpec{
			URL:       p.subscription.Spec.ProfileURL,
			Reference: ref,
		},
	}
}
