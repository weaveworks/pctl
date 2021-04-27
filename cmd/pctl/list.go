package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/profile"
	"k8s.io/kops/util/pkg/tables"
)

func listCmd() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "list profile subscriptions",
		UsageText: "pctl --kubeconfig=<kubeconfig-path> list",
		Action: func(c *cli.Context) error {
			cl, err := buildK8sClient(c.String("kubeconfig"))
			if err != nil {
				return err
			}
			profiles, err := profile.New(cl).List()
			if err != nil {
				return err
			}
			return printSubscriptions(profiles)
		},
	}
}

func printSubscriptions(subs []profile.SubscriptionDescription) error {
	table := tables.Table{}
	table.AddColumn("NAMESPACE", func(sub profile.SubscriptionDescription) string {
		return sub.Namespace
	})
	table.AddColumn("NAME", func(sub profile.SubscriptionDescription) string {
		return sub.Name
	})
	table.AddColumn("READY", func(sub profile.SubscriptionDescription) string {
		return sub.Ready
	})
	return table.Render(subs, os.Stdout, "NAMESPACE", "NAME", "READY")
}
