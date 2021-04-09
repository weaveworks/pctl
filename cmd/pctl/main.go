package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/catalog"
	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
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
