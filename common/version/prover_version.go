package version

import (
	"strconv"
	"strings"

	"github.com/scroll-tech/go-ethereum/log"
)

// CheckScrollProverVersion check the "scroll-prover" version, if it's different from the local one, return false
func CheckScrollProverVersion(proverVersion string) bool {
	// note the the version is in fact in the format of "tag-commit-scroll_prover-halo2",
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

// parseVersion takes a version string and returns its major, minor, and patch numbers.
func parseVersion(version string) (major, minor, patch int) {
	trimVersion := strings.TrimPrefix(version, "v")
	if trimVersion == version {
		log.Error("version does not start with v", "vesion", version)
		return 0, 0, 0
	}

	versionPart := strings.SplitN(trimVersion, "-", 2)[0]
	parts := strings.Split(versionPart, ".")
	if len(parts) != 3 {
		log.Error("invalid version format", "expected format", "v<major>.<minor>.<patch>", "got", version)
		return 0, 0, 0
	}

	var err error
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		log.Error("invalid major version", "value", parts[0], "error", err)
		return 0, 0, 0
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		log.Error("invalid minor version", "value", parts[1], "error", err)
		return 0, 0, 0
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		log.Error("invalid patch version", "value", parts[2], "error", err)
		return 0, 0, 0
	}

	return major, minor, patch
}

// CheckScrollRepoVersion checks if the proverVersion is at least the minimum required version.
func CheckScrollRepoVersion(proverVersion, minVersion string) bool {
	major1, minor1, patch1 := parseVersion(proverVersion)
	major2, minor2, patch2 := parseVersion(minVersion)

	if major1 != major2 {
		return major1 > major2
	}
	if minor1 != minor2 {
		return minor1 > minor2
	}
	return patch1 >= patch2
}
