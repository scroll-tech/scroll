package version

import (
	"strconv"
	"strings"
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

// CheckScrollProverVersionTag check the "scroll-prover" version's tag, if it's too old, return false
func CheckScrollProverVersionTag(proverVersion string) bool {
	// note the the version is in fact in the format of "tag-commit-scroll_prover-halo2",
	// so split-by-'-' length should be 4
	remote := strings.Split(proverVersion, "-")
	if len(remote) != 4 {
		return false
	}
	remoteTagNums := strings.Split(strings.TrimPrefix(remote[0], "v"), ".")
	if len(remoteTagNums) != 3 {
		return false
	}
	remoteTagMajor, err := strconv.Atoi(remoteTagNums[0])
	if err != nil {
		return false
	}
	remoteTagMinor, err := strconv.Atoi(remoteTagNums[1])
	if err != nil {
		return false
	}
	remoteTagPatch, err := strconv.Atoi(remoteTagNums[2])
	if err != nil {
		return false
	}
	if remoteTagMajor < 4 {
		return false
	}
	if remoteTagMinor == 1 && remoteTagPatch < 98 {
		return false
	}
	return true
}
