package version

var (
	// GitCommit is set with --ldflags "-X scroll-tech/go-roller/version.GitCommit=$(git rev-parse HEAD)"
	// TODO: upgrade to go 1.18+ and use build info, see: https://icinga.com/blog/2022/05/25/embedding-git-commit-information-in-go-binaries/
	GitCommit string
)
