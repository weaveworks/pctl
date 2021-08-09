package integration_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("pctl get", func() {
	Context("get catalog profiles", func() {
		It("returns the matching profiles", func() {
			Expect(pctl("get", "--catalog", "nginx")).To(ConsistOf(
				"PACKAGE CATALOG",
				"CATALOG/PROFILE                VERSION DESCRIPTION",
				"nginx-catalog/weaveworks-nginx v0.1.0  This installs nginx.",
				"nginx-catalog/weaveworks-nginx v0.1.1  This installs nginx.",
				"nginx-catalog/bitnami-nginx    v0.0.1  This installs nginx.",
				"nginx-catalog/nginx            v2.0.1  This installs nginx.",
				"nginx-catalog/some-other-nginx         This installs some other nginx.",
			))
		})

		It("returns all the catalog profiles", func() {
			Expect(pctl("get", "--catalog")).To(ConsistOf(
				"PACKAGE CATALOG",
				"CATALOG/PROFILE                VERSION DESCRIPTION",
				"nginx-catalog/weaveworks-nginx v0.1.0  This installs nginx.",
				"nginx-catalog/weaveworks-nginx v0.1.1  This installs nginx.",
				"nginx-catalog/bitnami-nginx    v0.0.1  This installs nginx.",
				"nginx-catalog/nginx            v2.0.1  This installs nginx.",
				"nginx-catalog/some-other-nginx         This installs some other nginx.",
			))
		})

		When("-o is set to json", func() {
			It("returns the matching profiles in json", func() {
				Expect(pctlWithRawOutput("get", "-o", "json", "--catalog", "nginx")).To(ContainSubstring(`{
    "tag": "weaveworks-nginx/v0.1.0",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "weaveworks-nginx/v0.1.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "bitnami-nginx/v0.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "bitnami-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "v2.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/nginx-profile",
    "name": "nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "catalogSource": "nginx-catalog",
    "name": "some-other-nginx",
    "description": "This installs some other nginx."
  }`))
			})

			It("returns all catalog profiles in json", func() {
				Expect(pctlWithRawOutput("get", "-o", "json", "--catalog")).To(ContainSubstring(`{
    "tag": "weaveworks-nginx/v0.1.0",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "weaveworks-nginx/v0.1.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "weaveworks-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "bitnami-nginx/v0.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/profiles-examples",
    "name": "bitnami-nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "tag": "v2.0.1",
    "catalogSource": "nginx-catalog",
    "url": "https://github.com/weaveworks/nginx-profile",
    "name": "nginx",
    "description": "This installs nginx.",
    "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
    "prerequisites": [
      "Kubernetes 1.18+"
    ]
  },
  {
    "catalogSource": "nginx-catalog",
    "name": "some-other-nginx",
    "description": "This installs some other nginx."
  }`))
			})
		})

		When("kubeconfig is incorrectly set", func() {
			It("returns a useful error", func() {
				Expect(pctlWithError("--kubeconfig=/non-existing/path/kubeconfig", "get", "nginx")).To(ContainElement(
					"failed to create config from kubeconfig path \"/non-existing/path/kubeconfig\": stat /non-existing/path/kubeconfig: no such file or directory",
				))
			})
		})
	})

	Context("installed profiles and catalog profiles with get", func() {
		var (
			installationName = "long-name-to-ensure-padding"
			ctx              = context.TODO()
			pInstallation    profilesv1.ProfileInstallation
		)

		BeforeEach(func() {
			namespace = uuid.New().String()
			createNamespace(namespace)

			profileURL := "https://github.com/weaveworks/profiles-examples"
			pInstallation = profilesv1.ProfileInstallation{
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
			Expect(kClient.Create(ctx, &pInstallation)).Should(Succeed())
		})

		AfterEach(func() {
			deleteNamespace(namespace)
		})

		It("returns all the installations and catalog profiles", func() {
			Expect(pctl("get")).To(ConsistOf(
				"INSTALLED PACKAGES",
				"NAMESPACE                            NAME                        SOURCE                                AVAILABLE UPDATES",
				fmt.Sprintf("%s long-name-to-ensure-padding nginx-catalog/weaveworks-nginx/v0.1.0 v0.1.1", namespace),
				"PACKAGE CATALOG",
				"CATALOG/PROFILE                VERSION DESCRIPTION",
				"nginx-catalog/weaveworks-nginx v0.1.0  This installs nginx.",
				"nginx-catalog/weaveworks-nginx v0.1.1  This installs nginx.",
				"nginx-catalog/bitnami-nginx    v0.0.1  This installs nginx.",
				"nginx-catalog/nginx            v2.0.1  This installs nginx.",
				"nginx-catalog/some-other-nginx         This installs some other nginx.",
			))
		})

		It("returns all the installations and catalog profiles with matching name", func() {
			Expect(kClient.Create(ctx, &profilesv1.ProfileInstallation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ProfileInstallation",
					APIVersion: "profileinstallations.weave.works/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bitnami-nginx",
					Namespace: namespace,
				},
				Spec: profilesv1.ProfileInstallationSpec{
					Source: &profilesv1.Source{
						URL: "https://github.com/weaveworks/profiles-examples",
						Tag: "bitnami-nginx/v0.0.1",
					},
					Catalog: &profilesv1.Catalog{
						Catalog: "nginx-catalog",
						Profile: "bitnami-nginx",
						Version: "v0.0.1",
					},
				},
			})).Should(Succeed())

			Expect(pctl("get", "bitnami-nginx")).To(ConsistOf(
				"INSTALLED PACKAGES",
				"NAMESPACE                            NAME          SOURCE                             AVAILABLE UPDATES",
				fmt.Sprintf("%s bitnami-nginx nginx-catalog/bitnami-nginx/v0.0.1 -", namespace),
				"PACKAGE CATALOG",
				"CATALOG/PROFILE             VERSION DESCRIPTION",
				"nginx-catalog/bitnami-nginx v0.0.1  This installs nginx.",
			))
		})
	})

	Context("installed profiles with get", func() {
		var (
			installationName = "long-name-to-ensure-padding"
			ctx              = context.TODO()
			pSub             profilesv1.ProfileInstallation
		)

		BeforeEach(func() {
			namespace = uuid.New().String()
			createNamespace(namespace)

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
						URL: "https://github.com/weaveworks/profiles-examples",
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
			deleteNamespace(namespace)
		})

		It("returns the installations", func() {
			Eventually(func() []string {
				return pctl("get", "--installed")
			}).Should(ConsistOf(
				"INSTALLED PACKAGES",
				"NAMESPACE                            NAME                        SOURCE                                AVAILABLE UPDATES",
				fmt.Sprintf("%s long-name-to-ensure-padding nginx-catalog/weaveworks-nginx/v0.1.0 v0.1.1", namespace),
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
				Eventually(func() []string {
					return pctl("get", "--installed")
				}).Should(ContainElements(
					"INSTALLED PACKAGES",
					"NAMESPACE                            NAME                        SOURCE                                AVAILABLE UPDATES",
					fmt.Sprintf("%s bitnami-profile             nginx-catalog/bitnami-nginx/v0.0.1    -", namespace),
					fmt.Sprintf("%s long-name-to-ensure-padding nginx-catalog/weaveworks-nginx/v0.1.0 v0.1.1", namespace),
				))
			})

			It("returns the installation matching name", func() {
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
				Eventually(func() []string {
					return pctl("get", "--installed", "bitnami-profile")
				}).Should(ContainElements(
					"INSTALLED PACKAGES",
					"NAMESPACE                            NAME            SOURCE                             AVAILABLE UPDATES",
					fmt.Sprintf("%s bitnami-profile nginx-catalog/bitnami-nginx/v0.0.1 -", namespace),
				))
			})
		})
	})

	Context("version", func() {
		When("version is used in the get command", func() {
			It("shows the right profile", func() {
				Expect(pctl("get", "--profile-version", "v0.1.0", "nginx-catalog/weaveworks-nginx")).To(ConsistOf(
					"Catalog       nginx-catalog",
					"Name          weaveworks-nginx",
					"Version       v0.1.0",
					"Description   This installs nginx.",
					"URL           https://github.com/weaveworks/profiles-examples",
					"Maintainer    weaveworks (https://github.com/weaveworks/profiles)",
					"Prerequisites Kubernetes 1.18+",
				))
			})
		})

		When("-o is set to json", func() {
			It("returns the profile info in json", func() {
				Expect(pctlWithRawOutput("get", "-o", "json", "--profile-version", "v0.1.0", "nginx-catalog/weaveworks-nginx")).To(ContainSubstring(`{
  "tag": "weaveworks-nginx/v0.1.0",
  "catalogSource": "nginx-catalog",
  "url": "https://github.com/weaveworks/profiles-examples",
  "name": "weaveworks-nginx",
  "description": "This installs nginx.",
  "maintainer": "weaveworks (https://github.com/weaveworks/profiles)",
  "prerequisites": [
    "Kubernetes 1.18+"
  ]
}`))
			})
		})

		When("a name argument is not provided correctly", func() {
			It("returns a useful error", func() {
				Expect(pctlWithError("get", "--profile-version", "v0.1.0", "test-profile")).To(ContainElement(
					"both catalog name and profile name must be provided example: pctl get catalog/weaveworks-nginx --version v0.1.0",
				))
			})
		})
	})
})
