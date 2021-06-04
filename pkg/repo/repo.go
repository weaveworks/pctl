package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"

	profilesv1 "github.com/weaveworks/profiles/api/v1alpha1"

	"github.com/weaveworks/pctl/pkg/git"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// seededRand provides a naive random number for a random prefix for the clone folder.
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// GetProfileDefinition returns a definition based on a url and a branch.
func GetProfileDefinition(repoURL, branch, path string, gitClient git.Git) (profilesv1.ProfileDefinition, error) {
	if _, err := url.Parse(repoURL); err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse repo URL %q: %w", repoURL, err)
	}

	if !strings.Contains(repoURL, "github.com") {
		return profilesv1.ProfileDefinition{}, errors.New("unsupported git provider, only github.com is currently supported")
	}

	// Add postfix so potential nested profiles don't clone into the same folder.
	px := postfix(6)
	tmp, err := ioutil.TempDir("", "get_profile_definition_clone_"+px)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to create temp folder for cloning repository: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			fmt.Println("failed to remove temp folder, please clean by hand: ", tmp)
		}
	}()

	if err := gitClient.Clone(repoURL, branch, tmp); err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to clone the repo: %w", err)
	}

	content, err := ioutil.ReadFile(filepath.Join(tmp, path, "profile.yaml"))
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("could not find file at cloned location: %w", err)
	}

	profile := profilesv1.ProfileDefinition{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096).Decode(&profile)
	if err != nil {
		return profilesv1.ProfileDefinition{}, fmt.Errorf("failed to parse profile: %w", err)
	}

	return profile, nil
}

// postfix generates a `length` long random string.
func postfix(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
