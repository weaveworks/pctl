package main

import (
	"fmt"
	"strings"

	"github.com/weaveworks/pctl/pkg/formatter"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/catalog"

	"github.com/weaveworks/pctl/pkg/client"
)

func listCmd() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "list installed profiles",
		UsageText: "pctl list",
		Action: func(c *cli.Context) error {
			cl, err := buildK8sClient(c.String("kubeconfig"))
			if err != nil {
				return err
			}
			options := client.ServiceOptions{
				KubeconfigPath: c.String("kubeconfig"),
				Namespace:      c.String("catalog-service-namespace"),
				ServiceName:    c.String("catalog-service-name"),
				ServicePort:    c.String("catalog-service-port"),
			}
			catalogClient, err := client.NewFromOptions(options)
			if err != nil {
				return err
			}
			data, err := catalog.List(cl, catalogClient)
			if err != nil {
				return err
			}
			var f formatter.Formatter
			f = formatter.NewTableFormatter()
			getter := listDataFunc(data)

			if c.String("output") == "json" {
				f = formatter.NewJSONFormatter()
				getter = func() interface{} { return data }
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
