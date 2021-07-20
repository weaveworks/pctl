package integration_test

import (
	"context"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

var _ = Describe("pctl list", func() {
	Context("list", func() {
		var (
			namespace        = "default"
			installationName = "long-name-to-ensure-padding"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileInstallation
		)

		BeforeEach(func() {
			profileURL := "https://github.com/weaveworks/profiles-examples"
			pSub = profilesv1.ProfileInstallation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileInstallation",
					APIVersion: "profileinstallations.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      installationName,
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL: profileURL,
						Tag: "weaveworks-nginx/v0.1.0",
					},
					Catalog: &profilesv1.Catalog{
						Catalog: "nginx-catalog",
						Profile: "weaveworks-nginx",
						Version: "v0.1.0",
					},
				},
			}
			Expect(kClient.Create(ctx, &pSub)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(kClient.Delete(ctx, &pSub)).Should(Succeed())
		})

		It("returns the installations", func() {
			listCmd := func() []string {
				cmd := exec.Command(binaryPath, "list")
				session, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				return strings.Split(string(session), "\n")
			}

			Eventually(listCmd).Should(ContainElements(
				"NAMESPACE\tNAME                       \tSOURCE                               \tAVAILABLE UPDATES ",
				"default  \tlong-name-to-ensure-padding\tnginx-catalog/weaveworks-nginx/v0.1.0\tv0.1.1           \t",
			))
		})

		When("there are no available updates", func() {
			It("returns the installations", func() {
				profileURL := "https://github.com/weaveworks/profiles-examples"
				bitnamiSub := profilesv1.ProfileInstallation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProfileInstallation",
						APIVersion: "profileinstallations.weave.works/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bitnami-profile",
						Namespace: namespace,
					},
					Spec: profilesv1.ProfileInstallationSpec{
						Source: &profilesv1.Source{
							URL: profileURL,
							Tag: "bitnami-nginx/v0.0.1",
						},
						Catalog: &profilesv1.Catalog{
							Catalog: "nginx-catalog",
							Profile: "bitnami-nginx",
							Version: "v0.0.1",
						},
					},
				}
				Expect(kClient.Create(ctx, &bitnamiSub)).Should(Succeed())
				listCmd := func() []string {
					cmd := exec.Command(binaryPath, "list")
					session, err := cmd.CombinedOutput()
					Expect(err).ToNot(HaveOccurred())
					return strings.Split(string(session), "\n")
				}

				Eventually(listCmd).Should(ContainElements(
					"NAMESPACE\tNAME                       \tSOURCE                               \tAVAILABLE UPDATES ",
					"default  \tbitnami-profile            \tnginx-catalog/bitnami-nginx/v0.0.1   \t-                \t",
					"default  \tlong-name-to-ensure-padding\tnginx-catalog/weaveworks-nginx/v0.1.0\tv0.1.1           \t",
				))
				Expect(kClient.Delete(ctx, &bitnamiSub)).Should(Succeed())
			})
		})
	})
})
