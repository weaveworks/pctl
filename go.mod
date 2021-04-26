module github.com/weaveworks/pctl

go 1.16

require (
	github.com/fluxcd/helm-controller/api v0.10.0
	github.com/jenkins-x/go-scm v1.8.1
	github.com/olekukonko/tablewriter v0.0.0-20210304033056-74c60be0ef68
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/weaveworks/profiles v0.0.0-20210415085322-61383f6e66ed
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/kops v1.19.0
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	// Used to get around some weird etcd/grpc incompatibilty
	google.golang.org/grpc => google.golang.org/grpc v1.29.0
	// Used to pin the k8s library versions regardless of what other dependencies enforce
	k8s.io/api => k8s.io/api v0.19.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.5
	k8s.io/apiserver => k8s.io/apiserver v0.19.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.5
	k8s.io/client-go => k8s.io/client-go v0.19.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.5
	k8s.io/code-generator => k8s.io/code-generator v0.19.5
	k8s.io/component-base => k8s.io/component-base v0.19.5
	k8s.io/cri-api => k8s.io/cri-api v0.19.5
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.5
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.5
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.5
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.5
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.5
	k8s.io/kubectl => k8s.io/kubectl v0.19.5
	k8s.io/kubelet => k8s.io/kubelet v0.19.5
	k8s.io/kubernetes => k8s.io/kubernetes v1.19.5
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.5
	k8s.io/metrics => k8s.io/metrics v0.19.5
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.5
)
