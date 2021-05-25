module github.com/weaveworks/pctl

go 1.16

require (
	github.com/fluxcd/helm-controller/api v0.10.1
	github.com/fluxcd/kustomize-controller/api v0.12.0
	github.com/fluxcd/source-controller/api v0.12.2
	github.com/google/uuid v1.2.0
	github.com/jenkins-x/go-scm v1.8.1
	github.com/olekukonko/tablewriter v0.0.0-20210304033056-74c60be0ef68
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/weaveworks/profiles v0.0.2
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.20.5
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/cli-utils v0.25.0
	sigs.k8s.io/controller-runtime v0.8.3
)
