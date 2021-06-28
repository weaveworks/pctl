package spike_test

import (
	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/pctl/spike"
	v1 "k8s.io/api/apps/v1"
)

var _ = Describe("Spike", func() {
	It("merges the objects", func() {
		original := &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(2),
				Paused:   false,
			},
		}

		userModified := &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(5),
				Paused:   false,
			},
		}
		latest := &v1.Deployment{
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(2),
				Paused:   true,
			},
		}

		returnValue, err := spike.Merge(original, userModified, latest)
		Expect(err).NotTo(HaveOccurred())
		Expect(*returnValue.(*v1.Deployment).Spec.Replicas).To(Equal(int32(5)))
		Expect(returnValue.(*v1.Deployment).Spec.Paused).To(Equal(true))
	})
})
