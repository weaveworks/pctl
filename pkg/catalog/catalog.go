package catalog

// CatalogClient makes requests to the catalog service
//go:generate counterfeiter -o fakes/fake_catalog_client.go . CatalogClient
type CatalogClient interface {
	DoRequest(path string, query map[string]string) ([]byte, error)
}

