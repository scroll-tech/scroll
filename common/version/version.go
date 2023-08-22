package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var tag = "v4.1.90"

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
	// Set default value for integration test.
	return "000000"
}()

// ZkVersion is commit-id of common/libzkp/impl/cargo.lock/scroll-prover and halo2, contacted by a "-"
// The default `000000-000000` is set for integration test, and will be overwritten by coordinator's & prover's actual compilations (see their Makefiles).
var ZkVersion = "000000-000000"

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
	// compare the `scroll_prover` version
	return remote[2] == local[2] || // libzkp v0.6.6
		remote[2] == "ccb3cd4" || // libzkp v0.6.5
		remote[2] == "8c439b1" // libzkp v0.6.2
}
