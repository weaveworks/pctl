package profile

import (
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Profile) makeGitRepository() *sourcev1.GitRepository {
	ref := &sourcev1.GitRepositoryRef{
		Branch: p.subscription.Spec.Branch,
	}
	if p.subscription.Spec.Tag != "" {
		ref = &sourcev1.GitRepositoryRef{
			Tag: p.subscription.Spec.Tag,
		}
	}
	return &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.makeGitRepoName(),
			Namespace: p.subscription.ObjectMeta.Namespace,
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

func (p *Profile) makeGitRepoName() string {
	repoParts := strings.Split(p.subscription.Spec.ProfileURL, "/")
	repoName := repoParts[len(repoParts)-1]
	if p.subscription.Spec.Tag != "" {
		parts := strings.Split(p.subscription.Spec.Tag, "/")
		return join(p.subscription.Name, repoName, parts[1])
	}
	branch := SanitiseBranchName(p.subscription.Spec.Branch)
	return join(p.subscription.Name, repoName, branch)
}
