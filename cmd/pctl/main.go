package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/profiles/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func main() {
	app := &cli.App{
		Flags: globalFlags(),
		Commands: []*cli.Command{
			searchCmd(),
			showCmd(),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func searchCmd() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "search for a profile",
		Action: func(c *cli.Context) error {
			catalogURL := c.String("catalog-url")
			if catalogURL == "" {
				return fmt.Errorf("--catalog-url or $PCTL_CATALOG_URL must be provided")
			}
			searchName := c.Args().First()
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
		Name:  "show",
		Usage: "display information about a profile",
		Action: func(c *cli.Context) error {
			catalogURL := c.String("catalog-url")
			if catalogURL == "" {
				return fmt.Errorf("--catalog-url or $PCTL_CATALOG_URL must be provided")
			}
			profileName := c.Args().First()
			fmt.Printf("retrieving information for profile %q:\n\n", profileName)
			profile, err := catalog.Show(catalogURL, profileName)
			if err != nil {
				return err
			}
			return printProfile(profile)
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

func printProfile(profile v1alpha1.ProfileDescription) error {
	out, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	return nil
}
