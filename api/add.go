package api

import (
	"errors"
	"fmt"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/git"
	"github.com/weaveworks/pctl/pkg/install"
	"github.com/weaveworks/pctl/pkg/log"
)

// AddProfileOpts defines settings for adding profiles.
type AddProfileOpts struct {
	Branch                 string
	CatalogClient          catalog.CatalogClient
	CatalogName            string
	ConfigMap              string
	InstallationDirectory  string
	GitClient              git.Git
	GitRepositoryName      string
	GitRepositoryNamespace string
	Namespace              string
	Path                   string
	ProfileName            string
	SubName                string
	URL                    string
	Version                string
}

// Validate checks settings of the opts which can or cannot be used together
// and provides a reasonable error if needed.
func (o AddProfileOpts) Validate() error {
	if o.URL == "" && o.ProfileName == "" && o.CatalogName == "" {
		return errors.New("please provide either url or profile name with catalog name")
	}
	if o.URL != "" && o.ProfileName != "" && o.CatalogName != "" {
		return errors.New("please provide either url or profile name with catalog name")
	}
	if o.ProfileName == "" && o.CatalogName != "" || o.ProfileName != "" && o.CatalogName == "" {
		return errors.New("both catalog name and profile name must be provided")
	}
	return nil
}

// AddProfile add runs the add part of the `add` command.
func AddProfile(opts AddProfileOpts) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	var source string
	if opts.URL != "" && opts.Path != "" {
		source = fmt.Sprintf("repository %s, path: %s and branch %s", opts.URL, opts.Path, opts.Branch)
	} else if opts.URL != "" && opts.Path == "" {
		source = fmt.Sprintf("repository %s and branch %s", opts.URL, opts.Branch)
	} else {
		source = fmt.Sprintf("catalog entry %s/%s/%s", opts.CatalogName, opts.ProfileName, opts.Version)
	}

	log.Actionf("generating profile installation from source: %s", source)
	installer := install.NewInstaller(install.Config{
		GitClient:        opts.GitClient,
		RootDir:          opts.InstallationDirectory,
		GitRepoNamespace: opts.GitRepositoryNamespace,
		GitRepoName:      opts.GitRepositoryName,
	})
	cfg := catalog.InstallConfig{
		Clients: catalog.Clients{
			CatalogClient: opts.CatalogClient,
			Installer:     installer,
		},
		Profile: catalog.Profile{
			ProfileConfig: catalog.ProfileConfig{
				CatalogName:   opts.CatalogName,
				ConfigMap:     opts.ConfigMap,
				Namespace:     opts.Namespace,
				Path:          opts.Path,
				ProfileBranch: opts.Branch,
				ProfileName:   opts.ProfileName,
				SubName:       opts.SubName,
				URL:           opts.URL,
				Version:       opts.Version,
			},
			GitRepoConfig: catalog.GitRepoConfig{
				Namespace: opts.GitRepositoryNamespace,
				Name:      opts.GitRepositoryName,
			},
		},
	}
	manager := &catalog.Manager{}
	return manager.Install(cfg)
}
