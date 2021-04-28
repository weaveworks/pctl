package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/subscription"
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
			profiles, err := subscription.NewManager(cl).List()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				fmt.Println("no profiles found")
				return nil
			}
			return printSubscriptions(profiles)
		},
	}
}

func printSubscriptions(subs []subscription.SubscriptionSummary) error {
	table := tables.Table{}
	table.AddColumn("NAMESPACE", func(sub subscription.SubscriptionSummary) string {
		return sub.Namespace
	})
	table.AddColumn("NAME", func(sub subscription.SubscriptionSummary) string {
		return sub.Name
	})
	table.AddColumn("READY", func(sub subscription.SubscriptionSummary) string {
		return sub.Ready
	})
	return table.Render(subs, os.Stdout, "NAMESPACE", "NAME", "READY")
}
