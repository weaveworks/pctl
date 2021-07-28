package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/catalog"
	"github.com/weaveworks/pctl/pkg/client"
	"github.com/weaveworks/pctl/pkg/formatter"
)

func getCmd() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "search for a profile",
		UsageText: "pctl get [--output table|json <QUERY> --installed --catalog --version --all] \n\n" +
			"   example: pctl get nginx",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				DefaultText: "table",
				Value:       "table",
				Usage:       "Output format. json|table",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Search all available profiles",
			},
			&cli.BoolFlag{
				Name:    "installed",
				Aliases: []string{"i"},
				Usage:   "Search all installed profiles",
			},
			&cli.BoolFlag{
				Name:    "catalog",
				Aliases: []string{"c"},
				Usage:   "Search all profiles in a catalog",
			},
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "display information about a profile from a catalog",
			},
		},
		Action: func(c *cli.Context) error {
			var profiles []profilesv1.ProfileCatalogEntry
			outFormat := c.String("output")
			manager := &catalog.Manager{}
			
			cl, err := buildK8sClient(c.String("kubeconfig"))
			if err != nil {
				return err
			}
			catalogClient, err := getCatalogClient(c)
			if err != nil {
				_ = cli.ShowCommandHelp(c, "get")
				return err
			}

			if c.Args().Len() > 0 && c.Bool("installed") {
				return getInstalledProfiles(c, cl, catalogClient)
			}

			if c.Args().Len() > 0 {
				profiles, err = manager.Search(catalogClient, "")
				if err != nil {
					return err
				}
			} else {
				searchName, catalogClient, err := parseArgs(c)
				if err != nil {
					_ = cli.ShowCommandHelp(c, "get")
					return err
				}

				// check if other flags are passed along with the searchname
				if c.Bool("catalog") {
					profiles, err = manager.Search(catalogClient, searchName)
					if err != nil {
						return err
					}
				} else if c.Bool("version") {
					profiles, err = manager.Search(catalogClient, searchName)
					if err != nil {
						return err
					}
				} else {
					// defaull get all installed and profiles from a catalog
					profiles, err = manager.Search(catalogClient, searchName)
					if err != nil {
						return err
					}
				}
			}

			return formatOutput(profiles, outFormat)
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

func formatOutput(profiles []profilesv1.ProfileCatalogEntry, outFormat string) error {
	if len(profiles) == 0 {
		fmt.Println("No profiles found")
		return nil
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
}

func formatInstalledProfilesOutput(data []catalog.ProfileData, outFormat string) error {
	if len(data) == 0 {
		fmt.Println("no profiles installed")
		return nil
	}

	var f formatter.Formatter
	f = formatter.NewTableFormatter()
	getter := listDataFunc(data)

	if outFormat == "json" {
		f = formatter.NewJSONFormatter()
		getter = func() interface{} { return data }
	}

	out, err := f.Format(getter)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func getInstalledProfiles(c *cli.Context, cl runtimeclient.Client, catalogClient *client.Client) error {
	manager := &catalog.Manager{}
	data, err := manager.List(cl, catalogClient)
	if err != nil {
		return err
	}

	outFormat := c.String("output")
	return formatInstalledProfilesOutput(data, outFormat)
}

func listDataFunc(data []catalog.ProfileData) func() interface{} {
	return func() interface{} {
		tc := formatter.TableContents{
			Headers: []string{"Namespace", "Name", "Source", "Available Updates"},
		}
		for _, d := range data {
			source := fmt.Sprintf("%s/%s/%s", d.Profile.Catalog, d.Profile.Profile, d.Profile.Version)
			if d.Profile.Catalog == "-" {
				source = fmt.Sprintf("%s:%s:%s", d.Profile.URL, d.Profile.Branch, d.Profile.Path)
			}
			tc.Data = append(tc.Data, []string{
				d.Profile.Namespace,
				d.Profile.Name,
				source,
				strings.Join(d.AvailableVersionUpdates, ","),
			})
		}
		return tc
	}
}
