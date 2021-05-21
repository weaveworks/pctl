package repo

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"
)

// HTTPClient defines an interface for HTTP get requests.
//go:generate counterfeiter -o fakes/fake_http_client.go . HTTPClient
type HTTPClient interface {
	Get(string) (*http.Response, error)
}

var httpClient HTTPClient = http.DefaultClient

// GetProfileDefinition returns a definition based on a url and a branch.
func GetProfileDefinition(repoURL, branch, path string) (profilesv1.ProfileDefinition, error) {
	fmt.Println("url, branch, path:", repoURL, branch, path)
	if _, err := url.Parse(repoURL); err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse repo URL %q: %w", repoURL, err)
	}

	if !strings.Contains(repoURL, "github.com") {
		return profilesv1.ProfileDefinition{}, errors.New("unsupported git provider, only github.com is currently supported")
	}

	profileURL := strings.Replace(repoURL, "github.com", "raw.githubusercontent.com", 1)
	profileURL = fmt.Sprintf("%s/%s/%s/profile.yaml", profileURL, branch, path)

	resp, err := httpClient.Get(profileURL)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to fetch profile: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to fetch profile: status code %d", resp.StatusCode)
	}

	profile := profilesv1.ProfileDefinition{}
	err = yaml.NewYAMLOrJSONDecoder(resp.Body, 4096).Decode(&profile)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}
