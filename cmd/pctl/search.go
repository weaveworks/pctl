package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/formatter"
)

func searchCmd() *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "search for a profile",
		UsageText: "pctl search [--output table|json] <QUERY>\n\n" +
			"   example: pctl search nginx",
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
			searchName, catalogClient, err := parseArgs(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "search")
				return err
			}

			profiles, err := catalog.Search(catalogClient, searchName)
			if err != nil {
				return err
			}
			outFormat := c.String("output")
			if outFormat == "table" {
				if len(profiles) == 0 {
					fmt.Printf("No profiles found matching: '%s'\n", searchName)
					return nil
				}
			}

			var f formatter.Formatter
			f = formatter.NewTableFormatter()
			getter := searchDataFunc(profiles)

			if outFormat == "json" {
				f = formatter.NewJSONFormatter()
				getter = func() interface{} { return profiles }
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

func searchDataFunc(profiles []profilesv1.ProfileCatalogEntry) func() interface{} {
	return func() interface{} {
		tc := formatter.TableContents{
			Headers: []string{"Catalog/Profile", "Version", "Description"},
		}
		for _, profile := range profiles {
			tc.Data = append(tc.Data, []string{
				fmt.Sprintf("%s/%s", profile.CatalogSource, profile.Name),
				profilesv1.GetVersionFromTag(profile.Tag),
				profile.Description,
			})
		}
		return tc
	}
}
