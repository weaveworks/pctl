package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/formatter"
)

func showCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Usage:     "display information about a profile",
		UsageText: "pctl [--kubeconfig=<kubeconfig-path>] show <CATALOG>/<PROFILE>[/<VERSION>]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				DefaultText: "table",
				Value:       "table",
				Usage:       "Output format. json|table",
			},
		},
		Action: func(c *cli.Context) error {
			profilePath, catalogClient, err := parseArgs(c)
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
			var catalogVersion string
			if len(parts) == 3 {
				catalogVersion = parts[2]
			}
			profile, err := catalog.Show(catalogClient, catalogName, profileName, catalogVersion)
			if err != nil {
				return err
			}

			var f formatter.Formatter
			f = formatter.NewTableFormatter()
			getter := showDataFunc(profile)

			if c.String("output") == "json" {
				f = formatter.NewJSONFormatter()
				getter = func() interface{} { return profile }
			}

			out, err := f.Format(getter)
			if err != nil {
				return err
			}

			fmt.Println(out)
			return nil
		},
	}
}

func showDataFunc(profile profilesv1.ProfileDescription) func() interface{} {
	return func() interface{} {
		return formatter.TableContents{
			Data: [][]string{
				{"Catalog", profile.CatalogSource},
				{"Name", profile.Name},
				{"Version", profile.Version},
				{"Description", profile.Description},
				{"URL", profile.URL},
				{"Maintainer", profile.Maintainer},
				{"Prerequisites", strings.Join(profile.Prerequisites, ", ")},
			},
		}
	}
}
