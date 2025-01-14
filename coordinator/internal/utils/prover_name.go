package utils

import "strings"

// IsExternalProverNameMatch checks if the local and remote external prover names belong to the same provider.
// It returns true if they do, otherwise false.
func IsExternalProverNameMatch(localName, remoteName string) bool {
	local := strings.Split(localName, "_")
	remote := strings.Split(remoteName, "_")

	if len(local) < 3 || len(remote) < 3 {
		return false
	}

	// note the name of cloud prover is in the format of "cloud_prover_{provider-name}_index"
	return local[0] == remote[0] && local[1] == remote[1] && local[2] == remote[2]
}
