package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/formatter"
	"github.com/weaveworks/pctl/pkg/subscription"
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
			profiles, err := subscription.NewManager(cl).List()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				fmt.Println("no profiles found")
				return nil
			}

			var f formatter.Formatter
			f = formatter.NewTableFormatter()
			getter := listDataFunc(profiles)

			if c.String("output") == "json" {
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

func listDataFunc(profiles []subscription.SubscriptionSummary) func() interface{} {
	return func() interface{} {
		tc := formatter.TableContents{
			Headers: []string{"Namespace", "Name", "Source"},
		}
		for _, profile := range profiles {
			source := fmt.Sprintf("%s/%s/%s", profile.Catalog, profile.Profile, profile.Version)
			if profile.Catalog == "-" {
				source = fmt.Sprintf("%s:%s:%s", profile.URL, profile.Branch, profile.Path)
			}
			tc.Data = append(tc.Data, []string{
				profile.Namespace,
				profile.Name,
				source,
			})
		}
		return tc
	}
}
