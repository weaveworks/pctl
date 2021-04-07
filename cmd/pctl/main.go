package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/weaveworks/pctl/pkg/catalog"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "search",
				Usage: "search for a profle",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "catalog-url",
						Usage:   "Catalog url",
						EnvVars: []string{"PCTL_CATALOG_URL"},
					},
				},
				Action: func(c *cli.Context) error {
					catalogURL := c.String("catalog-url")
					if catalogURL == "" {
						return fmt.Errorf("--catalog-url or $PCTL_CATALOG_URL must be provided")
					}
					profileName := c.Args().First()
					fmt.Printf("searching for profiles matching %q:\n", profileName)
					profiles, err := catalog.Search(catalogURL, profileName)
					if err != nil {
						return err
					}
					for _, profile := range profiles {
						fmt.Printf("%s: %s\n", profile.Name, profile.Description)
					}
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
