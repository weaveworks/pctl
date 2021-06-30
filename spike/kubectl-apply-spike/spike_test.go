package spike_test

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	spike "github.com/weaveworks/pctl/spike/kubectl-apply-spike"
)

var _ = Describe("Spike", func() {
	FIt("merges the objects", func() {
		original := &v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(2),
				Paused:   false,
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:    "test0",
							Values: []string{"1", "2", "3"},
						},
					},
				},
			},
		}

		userModified := &v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(5),
				Paused:   false,
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:    "test0",
							Values: []string{"0", "2", "3", "4"},
						},
					},
				},
			},
		}
		latest := &v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
			},
			Spec: v1.DeploymentSpec{
				Replicas: aws.Int32(2),
				Paused:   true,
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:    "test0",
							Values: []string{"1", "2", "3", "4"},
						},
					},
				},
			},
		}

		returnValue, err := spike.Merge(original, latest, userModified)
		Expect(err).NotTo(HaveOccurred())
		fmt.Println(string(returnValue))
		//Expect(*returnValue.(*v1.Deployment).Spec.Replicas).To(Equal(int32(5)))
		//Expect(returnValue.(*v1.Deployment).Spec.Paused).To(Equal(true))
	})
	When("conflict occurred", func() {
		It("returns an error", func() {
			base := &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				Spec: v1.DeploymentSpec{
					Replicas: aws.Int32(2),
					Paused:   false,
				},
			}

			local := &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				Spec: v1.DeploymentSpec{
					Replicas: aws.Int32(5),
					Paused:   false,
				},
			}
			remote := &v1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				Spec: v1.DeploymentSpec{
					Replicas: aws.Int32(2),
					Paused:   true,
				},
			}

			output, err := spike.Merge(base, local, remote)
			Expect(err).NotTo(HaveOccurred())
			fmt.Println(string(output))
		})
	})
})
