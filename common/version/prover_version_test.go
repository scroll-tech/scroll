package version

import (
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/log"
)

func TestMain(m *testing.M) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	m.Run()
}

func TestCheckScrollProverVersion(t *testing.T) {
	tests := []struct {
		proverVersion string
		want          bool
	}{
		{Version, true},
		{"tag-commit-111111-000000", false},
		{"incorrect-format", false},
		{"tag-commit-222222-111111", false},
	}

	for _, tt := range tests {
		if got := CheckScrollProverVersion(tt.proverVersion); got != tt.want {
			t.Errorf("CheckScrollProverVersion(%q) = %v, want %v", tt.proverVersion, got, tt.want)
		}
	}
}

func TestCheckScrollRepoVersion(t *testing.T) {
	tests := []struct {
		proverVersion string
		minVersion    string
		want          bool
	}{
		{"v1.2.3-commit-111111-000000", "v1.2.3-alpha", true},
		{"v1.2.3-patch-commit-111111-000000", "v1.2.2", true},
		{"v1.0.0", "v1.0.0-alpha", true},
		{"v1.2.2", "v1.2.3", false},
		{"v2.0.0", "v1.9.9", true},
		{"v0.9.0", "v1.0.0", false},
		{"v9.9.9", "v10.0.0-alpha", false},
		{"v4.1.98-aaa-bbb-ccc", "v999.0.0", false},
	}

	for _, tt := range tests {
		if got := CheckScrollRepoVersion(tt.proverVersion, tt.minVersion); got != tt.want {
			t.Errorf("CheckScrollRepoVersion(%q, %q) = %v, want %v", tt.proverVersion, tt.minVersion, got, tt.want)
		}
	}
}
