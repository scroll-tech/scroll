package version

import (
	"fmt"
	"runtime/debug"
)

var tag = "prealpha-v7.2"

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

// Version denote the version of scroll protocol, including the l2geth, relayer, coordinator, roller, contracts and etc.
var Version = fmt.Sprintf("%s-%s", tag, commit)
