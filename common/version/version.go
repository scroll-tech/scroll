package version

import (
	"fmt"
	"runtime/debug"
)

var tag = "v4.0.34"

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

// ZkVersion is commit-id of common/libzkp/impl/cargo.lock/scroll-prover
var ZkVersion string

// Version denote the version of scroll protocol, including the l2geth, relayer, coordinator, prover, contracts and etc.
var Version = fmt.Sprintf("%s-%s-%s", tag, commit, ZkVersion)
