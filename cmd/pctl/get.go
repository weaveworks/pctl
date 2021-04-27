package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/weaveworks/pctl/pkg/subscription"
)

func getCmd() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "get a profile Subscription",
		UsageText: "pctl --kubeconfig=<kubeconfig-path> get --name my-sub --namespace default",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "The name of the subscription.",
			},
			&cli.StringFlag{
				Name:        "namespace",
				DefaultText: "default",
				Value:       "default",
				Usage:       "The namespace the subscrption is in",
			},
		},
		Action: func(c *cli.Context) error {
			namespace := c.String("namespace")
			name := c.String("name")
			if name == "" {
				return fmt.Errorf("subscrption name must be provided")
			}
			cl, err := buildK8sClient(c.String("kubeconfig"))
			if err != nil {
				return err
			}
			profile, err := subscription.NewManager(cl).Get(namespace, name)
			if err != nil {
				return err
			}
			printSubscription(profile)
			return nil
		},
	}
}

func printSubscription(sub subscription.SubscriptionSummary) {
	fmt.Printf(`Subscription: %s
Namespace: %s
Ready: %s
Reason:
 - %s
`, sub.Name, sub.Namespace, sub.Ready, sub.Reason)
}
