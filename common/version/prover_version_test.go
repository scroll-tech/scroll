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

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
	}{
		{"v1.2.3-commit-111111-000000", 1, 2, 3},
		{"v0.10.0-patch-commit-111111-000000", 0, 10, 0},
		{"v2.0.1-alpha", 2, 0, 1},
		{"v10.0.0", 10, 0, 0},
		{"v1.0", 0, 0, 0}, // Invalid format
		{"v..", 0, 0, 0},  // Invalid format
	}

	for _, tt := range tests {
		gotMajor, gotMinor, gotPatch := parseVersion(tt.version)
		if gotMajor != tt.wantMajor || gotMinor != tt.wantMinor || gotPatch != tt.wantPatch {
			t.Errorf("parseVersion(%q) = %v, %v, %v, want %v, %v, %v", tt.version, gotMajor, gotMinor, gotPatch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
		}
	}
}

func TestCheckScrollRepoVersion(t *testing.T) {
	tests := []struct {
		proverVersion string
		minVersion    string
		want          bool
	}{
		{"v1.2.3-commit-111111-000000", "v1.2.3", true},
		{"v1.2.3-patch-commit-111111-000000", "v1.2.2", true},
		{"v1.0.0-alpha", "v1.0.0", true},
		{"v1.2.2", "v1.2.3", false},
		{"v2.0.0", "v1.9.9", true},
		{"v0.9.0", "v1.0.0", false},
		{"v9.9.9", "v10.0.0", false},
		{"v4.1.98-aaa-bbb-ccc", "v999.0.0", false},
	}

	for _, tt := range tests {
		if got := CheckScrollRepoVersion(tt.proverVersion, tt.minVersion); got != tt.want {
			t.Errorf("CheckScrollRepoVersion(%q, %q) = %v, want %v", tt.proverVersion, tt.minVersion, got, tt.want)
		}
	}
}
