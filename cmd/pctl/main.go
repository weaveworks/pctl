package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/writer"
)

func main() {
	app := &cli.App{
		Usage: "A cli tool for interacting with profiles",
		Flags: globalFlags(),
		Commands: []*cli.Command{
			searchCmd(),
			showCmd(),
			installCmd(),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func searchCmd() *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "search for a profile",
		UsageText: "pctl --catalog-url <URL> search <QUERY>",
		Action: func(c *cli.Context) error {
			searchName, catalogURL, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "show")
				return err
			}

			fmt.Printf("searching for profiles matching %q:\n", searchName)
			profiles, err := catalog.Search(catalogURL, searchName)
			if err != nil {
				return err
			}
			for _, profile := range profiles {
				fmt.Printf("%s: %s\n", profile.Name, profile.Description)
			}
			return nil
		},
	}
}

func showCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "display information about a profile",
		UsageText: "pctl --catalog-url <URL> show <CATALOG>/<PROFILE>",
		Action: func(c *cli.Context) error {
			profilePath, catalogURL, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "show")
				return err
			}

			parts := strings.Split(profilePath, "/")
			if len(parts) < 2 {
				_ = cli.ShowCommandHelp(c, "show")
				return errors.New("both catalog name and profile name must be provided")
			}
			catalogName, profileName := parts[0], parts[1]

			fmt.Printf("retrieving information for profile %s/%s:\n\n", catalogName, profileName)
			profile, err := catalog.Show(catalogURL, catalogName, profileName)
			if err != nil {
				return err
			}
			return printProfile(profile)
		},
	}
}

func installCmd() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "generate a profile subscription for a profile in a catalog",
		UsageText: "pctl --catalog-url <URL> install --subscription-name pctl-profile --namespace default --branch main --config-secret configmap-name --out profile_subscription.yaml <CATALOG>/<PROFILE>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "subscription-name",
				DefaultText: "pctl-profile",
				Value:       "pctl-profile",
				Usage:       "The name of the subscription.",
			},
			&cli.StringFlag{
				Name:        "namespace",
				DefaultText: "default",
				Value:       "default",
				Usage:       "The namespace to use for generating resources.",
			},
			&cli.StringFlag{
				Name:        "branch",
				Value:       "main",
				DefaultText: "main",
				Usage:       "The branch to use on the repository in which the profile is.",
			},
			&cli.StringFlag{
				Name:  "config-secret",
				Value: "",
				Usage: "The name of the ConfigMap which contains values for this profile.",
			},
			&cli.StringFlag{
				Name:        "out",
				Value:       "profile_subscription.yaml",
				DefaultText: "profile_subscription.yaml",
				Usage:       "Filename to use for the generated content.",
			},
		},
		Action: func(c *cli.Context) error {
			profilePath, catalogURL, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "install")
				return err
			}

			branch := c.String("branch")
			subName := c.String("subscription-name")
			namespace := c.String("namespace")
			configValues := c.String("config-secret")
			filename := c.String("out")

			parts := strings.Split(profilePath, "/")
			if len(parts) < 2 {
				_ = cli.ShowCommandHelp(c, "install")
				return errors.New("both catalog name and profile name must be provided")
			}
			catalogName, profileName := parts[0], parts[1]

			fmt.Printf("generating subscription for profile %s/%s:\n\n", catalogName, profileName)
			var w writer.Writer = &writer.FileWriter{Filename: filename}
			cfg := catalog.InstallConfig{
				Branch:      branch,
				CatalogName: catalogName,
				CatalogURL:  catalogURL,
				ConfigMap:   configValues,
				Namespace:   namespace,
				ProfileName: profileName,
				SubName:     subName,
				Writer:      w,
			}
			return catalog.Install(cfg)
		},
	}
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "catalog-url",
			Usage:   "Catalog url",
			EnvVars: []string{"PCTL_CATALOG_URL"},
		},
	}
}

func parseArgs(c *cli.Context) (string, string, error) {
	catalogURL := c.String("catalog-url")
	if catalogURL == "" {
		return "", "", fmt.Errorf("--catalog-url or $PCTL_CATALOG_URL must be provided")
	}
	if c.Args().Len() < 1 {
		return "", "", fmt.Errorf("argument must be provided")
	}
	return c.Args().First(), catalogURL, nil
}

func printProfile(profile profilesv1.ProfileDescription) error {
	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
