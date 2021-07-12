package version

import (
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// ParsePctlVersion parses the an pctl version as semver while ignoring
// extra build metadata
func ParsePctlVersion(raw string) (semver.Version, error) {
	// We don't want any extra info from the version
	semverVersion := strings.Split(raw, ExtraSep)[0]
	v, err := semver.ParseTolerant(semverVersion)
	if err != nil {
		return v, errors.Wrapf(err, "unexpected error parsing pctl version %q", raw)
	}
	return v, nil
}
