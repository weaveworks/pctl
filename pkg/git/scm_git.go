package git

import (
	"context"
	"fmt"
	"os"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
)

const githubTokenEnvVar = "GITHUB_TOKEN"

// SCMClient defines the ability to create a pull request on a remote repository.
//go:generate counterfeiter -o fakes/fake_scm.go . SCMClient
type SCMClient interface {
	CreatePullRequest() error
}

// SCMConfig defines configuration for the SCM Client that is needed to create a pull request.
type SCMConfig struct {
	Branch string
	Base   string
	Repo   string
	Client *scm.Client
}

// Client defines a client which uses a real implementation to create pull requests.
type Client struct {
	SCMConfig
}

// NewClient returns a real client.
func NewClient(cfg SCMConfig) (*Client, error) {
	if cfg.Client == nil {
		githubToken := os.Getenv(githubTokenEnvVar)
		if githubToken == "" {
			return nil, fmt.Errorf("failed to create scm client: %s not set", githubTokenEnvVar)
		}
		c, err := factory.NewClient("github", "", githubToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create scm client: %w", err)
		}
		cfg.Client = c
	}
	return &Client{
		SCMConfig: cfg,
	}, nil
}

// CreatePullRequest will create a pull request.
func (r *Client) CreatePullRequest() error {
	fmt.Println("Creating pull request with : ", r.Repo, r.Base, r.Branch)
	ctx := context.Background()
	request, _, err := r.Client.PullRequests.Create(ctx, r.Repo, &scm.PullRequestInput{
		Title: "PCTL Generated Profile Resource Update",
		Head:  r.Branch,
		Base:  r.Base,
	})
	if err != nil {
		return fmt.Errorf("error while creating pr: %w", err)
	}
	fmt.Printf("PR created with number: %d and URL: %s\n", request.Number, request.Link)
	return nil
}
