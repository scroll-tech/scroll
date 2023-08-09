package version

import (
	"fmt"
	"runtime/debug"
)

var tag = "v4.1.23"

var commit = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				value := setting.Value
				if len(value) >= 7 {
					return value[:7]
				}
				return value
			}
		}
	}
	return ""
}()

// ZkVersion is commit-id of common/libzkp/impl/cargo.lock/scroll-prover and halo2, cancated by a "-"
var ZkVersion string

// Version denote the version of scroll protocol, including the l2geth, relayer, coordinator, prover, contracts and etc.
var Version = fmt.Sprintf("%s-%s-%s", tag, commit, ZkVersion)

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
	return remote[3] == local[3]
}
