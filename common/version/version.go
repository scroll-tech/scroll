package version

import (
	"fmt"
	"runtime/debug"
)

var tag = "prealpha-v9.4"

var commit = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				value := setting.Value
				if len(value) >= 8 {
					return value[:8]
				}
				return value
			}
		}
	}
	return ""
}()

// ZK_VERSION is commit-id of common/libzkp/impl/cargo.lock/scroll-zkevm
var ZK_VERSION string

// Version denote the version of scroll protocol, including the l2geth, relayer, coordinator, roller, contracts and etc.
var Version = fmt.Sprintf("%s-%s-%s", tag, commit, ZK_VERSION)
