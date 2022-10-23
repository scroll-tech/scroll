package version

import (
	"fmt"
	"runtime/debug"
)

var tag = "prealpha-v3.0"

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

var Version = fmt.Sprintf("%s-%s", tag, commit)
