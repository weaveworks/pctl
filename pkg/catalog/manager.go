package catalog

import profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

//go:generate counterfeiter -o fakes/fake_catalog_manager.go . CatalogManager
type CatalogManager interface {
	Show(CatalogClient, string, string, string) (profilesv1.ProfileCatalogEntry, error)
	Install(InstallConfig) error
}

type Manager struct{}
