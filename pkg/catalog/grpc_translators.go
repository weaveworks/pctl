package catalog

import profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

type grpcProfileCatalogEntry struct {
	Item profilesv1.ProfileCatalogEntry
}

type grpcProfileCatalogEntryList struct {
	Items []profilesv1.ProfileCatalogEntry
}
