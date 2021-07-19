package catalog

import (
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate counterfeiter -o fakes/fake_catalog_manager.go . CatalogManager
type CatalogManager interface {
	Show(CatalogClient, string, string, string) (profilesv1.ProfileCatalogEntry, error)
	Search(CatalogClient, string) ([]profilesv1.ProfileCatalogEntry, error)
	Install(InstallConfig) error
	List(runtimeclient.Client, CatalogClient) ([]ProfileData, error)
}

type Manager struct{}
