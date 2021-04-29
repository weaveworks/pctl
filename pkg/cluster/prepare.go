package cluster

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/fluxcd/pkg/untar"
	"github.com/weaveworks/pctl/pkg/runner"
)

const (
	kubectlCmd = "kubectl"
)

// Fetcher will download a manifest tar file from a remote repository.
type Fetcher struct {
	Client *http.Client
}

// Applier applies the previously generated manifest files.
type Applier struct {
	Runner runner.Runner
}

// Preparer will prepare an environment.
type Preparer struct {
	PrepConfig
	Fetcher *Fetcher
	Applier *Applier
}

// PrepConfig defines configuration options for prepare.
type PrepConfig struct {
	// BaseURL is given even one would like to download manifests from a fork
	// or a test repo.
	BaseURL     string
	Location    string
	Version     string
	KubeContext string
	KubeConfig  string
	DryRun      bool
}

// NewPreparer creates a preparer with set dependencies ready to be used.
func NewPreparer(cfg PrepConfig) (*Preparer, error) {
	tmp, err := ioutil.TempDir("", "pctl-manifests")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp folder for manifest files: %w", err)
	}
	cfg.Location = tmp
	return &Preparer{
		PrepConfig: cfg,
		Fetcher: &Fetcher{
			Client: http.DefaultClient,
		},
		Applier: &Applier{
			Runner: &runner.CLIRunner{},
		},
	}, nil
}

// Prepare will prepare an environment with everything that is needed to run profiles.
func (p *Preparer) Prepare() error {
	defer func() {
		if err := os.RemoveAll(p.Location); err != nil {
			fmt.Printf("failed to remove temporary folder at location: %s. Please clean manually.", p.Location)
		}
	}()
	if err := p.Fetcher.Fetch(context.Background(), p.BaseURL, p.Version, p.Location); err != nil {
		return err
	}
	return p.Applier.Apply(p.Location, p.KubeContext, p.KubeConfig, p.DryRun)
}

// Fetch the latest or a version of the released manifest files for profiles.
func (f *Fetcher) Fetch(ctx context.Context, url, version, dir string) error {
	ghURL := fmt.Sprintf("%s/latest/download/manifests.tar.gz", url)
	if strings.HasPrefix(version, "v") {
		ghURL = fmt.Sprintf("%s/download/%s/manifests.tar.gz", url, version)
	}

	req, err := http.NewRequest("GET", ghURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := f.Client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			fmt.Println("Failed to close body reader.")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, status: %s", ghURL, resp.Status)
	}

	if _, err = untar.Untar(resp.Body, dir); err != nil {
		return fmt.Errorf("failed to untar manifests.tar.gz from %s, error: %w", ghURL, err)
	}

	return nil
}

// Apply applies the fetched manifest files to a cluster.
func (a *Applier) Apply(folder string, kubeContext string, kubeConfig string, dryRun bool) error {
	kubectlArgs := []string{"apply", "-f", folder}
	if dryRun {
		kubectlArgs = append(kubectlArgs, "--dry-run=client", "--output=yaml")
	}
	if kubeContext != "" {
		kubectlArgs = append(kubectlArgs, "--context="+kubeContext)
	}
	if kubeConfig != "" {
		kubectlArgs = append(kubectlArgs, "--kubeconfig="+kubeConfig)
	}
	if _, err := a.Runner.Run(kubectlCmd, kubectlArgs...); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// In a follow up ticket, make this wait for all the possible resources to be condition=available.
	fmt.Println("install finished")
	return nil
}
