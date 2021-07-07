module github.com/weaveworks/pctl

go 1.16

require (
	github.com/fluxcd/helm-controller/api v0.11.1
	github.com/fluxcd/kustomize-controller/api v0.13.2
	github.com/fluxcd/source-controller/api v0.15.3
	github.com/google/uuid v1.2.0
	github.com/jenkins-x/go-scm v1.9.2
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/otiai10/copy v1.6.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/weaveworks/profiles v0.0.12
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	knative.dev/pkg v0.0.0-20210412173742-b51994e3b312
	sigs.k8s.io/cli-utils v0.25.1-0.20210608181808-f3974341173a
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/kustomize/api v0.8.10
)

// pin kustomize to v4.1.3
replace (
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.8.10
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.10.21
)
