module github.com/weaveworks/pctl

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/dave/jennifer v1.4.1
	github.com/fluxcd/helm-controller/api v0.11.2
	github.com/fluxcd/kustomize-controller/api v0.14.1
	github.com/fluxcd/pkg/apis/meta v0.10.1
	github.com/fluxcd/pkg/runtime v0.12.1
	github.com/fluxcd/pkg/version v0.1.0
	github.com/fluxcd/source-controller/api v0.16.0
	github.com/google/uuid v1.3.0
	github.com/jenkins-x/go-scm v1.10.10
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/otiai10/copy v1.6.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/weaveworks/profiles v0.2.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.1
	knative.dev/pkg v0.0.0-20210412173742-b51994e3b312
	sigs.k8s.io/cli-utils v0.25.1-0.20210608181808-f3974341173a
	sigs.k8s.io/controller-runtime v0.10.0
	sigs.k8s.io/kustomize/api v0.9.0
	sigs.k8s.io/yaml v1.2.0
)

// pin kustomize to v4.1.3
replace (
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.8.10
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.10.21
)
