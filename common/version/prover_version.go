package version

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/scroll-tech/go-ethereum/log"
)

// CheckScrollProverVersion check the "scroll-prover" version, if it's different from the local one, return false
func CheckScrollProverVersion(proverVersion string) bool {
	// note the version is in fact in the format of "tag-commit-scroll_prover-halo2",
	// so split-by-'-' length should be 4
	remote := strings.Split(proverVersion, "-")
	if len(remote) != 4 {
		return false
	}
	local := strings.Split(Version, "-")
	if len(local) != 4 {
		return false
	}
	// compare the `scroll_prover` version
	return remote[2] == local[2]
}

// CheckScrollRepoVersion checks if the proverVersion is at least the minimum required version.
func CheckScrollRepoVersion(proverVersion, minVersion string) bool {
	c, err := semver.NewConstraint(">= " + minVersion + "-0")
	if err != nil {
		log.Error("failed to initialize constraint", "minVersion", minVersion, "error", err)
		return false
	}

	v, err := semver.NewVersion(proverVersion + "-z")
	if err != nil {
		log.Error("failed to parse version", "proverVersion", proverVersion, "error", err)
		return false
	}

	return c.Check(v)
}
