package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/writer"
)

// InstallConfig defines parameters for the installation call.
type InstallConfig struct {
	Branch      string
	CatalogName string
	CatalogURL  string
	ConfigMap   string
	Namespace   string
	ProfileName string
	SubName     string
	Writer      writer.Writer
}

// Install using the catalog at catalogURL and a profile matching the provided profileName generates a profile subscription
// writing it out with the provided profile subscription writer.
func Install(cfg InstallConfig) error {
	u, err := url.Parse(cfg.CatalogURL)
	if err != nil {
		return fmt.Errorf("failed to parse url %q: %w", cfg.CatalogURL, err)
	}

	u.Path = fmt.Sprintf("profiles/%s/%s", cfg.CatalogName, cfg.ProfileName)
	resp, err := doRequest(u, nil)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close the response body from profile show with error: %v/n", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("unable to find profile `%s` in catalog `%s`", cfg.ProfileName, cfg.CatalogName)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := profilesv1.ProfileDescription{}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return fmt.Errorf("failed to parse profile: %w", err)
	}

	subscription := profilesv1.ProfileSubscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ProfileSubscription",
			APIVersion: "weave.works/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SubName,
			Namespace: cfg.Namespace,
		},
		Spec: profilesv1.ProfileSubscriptionSpec{
			ProfileURL: profile.URL,
			Branch:     cfg.Branch,
		},
	}
	if cfg.ConfigMap != "" {
		subscription.Spec.ValuesFrom = []helmv2.ValuesReference{
			{
				Kind:      "ConfigMap",
				Name:      cfg.SubName + "-values",
				ValuesKey: cfg.ConfigMap,
			},
		}
	}
	if err := cfg.Writer.Output(&subscription); err != nil {
		return fmt.Errorf("failed to output subscription information: %w", err)
	}
	return nil
}
