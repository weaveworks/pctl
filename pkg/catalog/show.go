package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// ShowOptions provides options to the show command.
type ShowOptions struct {
	catalogName    string
	profileName    string
	catalogVersion string
}

// ShowOpts is the functional options for Show
type ShowOpts func(option *ShowOptions)

// WithCatalogName sets up catalog name for Show
func WithCatalogName(name string) ShowOpts {
	return func(option *ShowOptions) {
		option.catalogName = name
	}
}

// WithProfileName sets up the profile name for Show
func WithProfileName(name string) ShowOpts {
	return func(option *ShowOptions) {
		option.profileName = name
	}
}

// WithCatalogVersion sets up catalog version for Show
func WithCatalogVersion(version string) ShowOpts {
	return func(option *ShowOptions) {
		option.catalogVersion = version
	}
}

// Show queries the catalog at catalogURL for a profile matching the provided profileName
func Show(catalogClient CatalogClient, opts ...ShowOpts) (profilesv1.ProfileDescription, error) {
	options := &ShowOptions{}
	for _, o := range opts {
		o(options)
	}
	u, err := url.Parse("/profiles")
	if err != nil {
		return profilesv1.ProfileDescription{}, err
	}
	u.Path = path.Join(u.Path, options.catalogName, options.profileName, options.catalogVersion)
	data, code, err := catalogClient.DoRequest(u.String(), nil)
	if err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to do request: %w", err)
	}

	if code != http.StatusOK {
		if code == http.StatusNotFound {
			return profilesv1.ProfileDescription{},
				fmt.Errorf("unable to find profile %q in catalog %q (with version if provided: %s)",
					options.profileName, options.catalogName, options.catalogVersion)
		}
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to fetch profile from catalog, status code %d", code)
	}

	var profile profilesv1.ProfileDescription
	if err := json.Unmarshal(data, &profile); err != nil {
		return profilesv1.ProfileDescription{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
