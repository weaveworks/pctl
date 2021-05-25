package profile

import (
	"regexp"
	"strings"
)

var invalid = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// SanitiseBranchName makes the branch domain name friendly.
func SanitiseBranchName(branch string) string {
	sanitized := invalid.ReplaceAllString(branch, "")
	return strings.ReplaceAll(sanitized, "_", "-")
}
