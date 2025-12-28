package utils

import (
	"errors"
	"strings"
)

const (
	DomainOffset   = 6
	STFVersionMask = (1 << DomainOffset) - 1
)

// version get the version for the chain instance
func Version(hardForkName string, ValidiumMode bool) (uint8, error) {

	var domain, stfVersion uint8

	if ValidiumMode {
		domain = 1
		stfVersion = 1
	} else {
		domain = 0
		switch canonicalName := strings.ToLower(hardForkName); canonicalName {
		case "euclidv1":
			stfVersion = 6
		case "euclidv2":
			stfVersion = 7
		case "feynman":
			stfVersion = 8
		case "galileo":
			stfVersion = 9
		case "galileov2":
			stfVersion = 10
		default:
			return 0, errors.New("unknown fork name " + canonicalName)
		}
	}

	return (domain << DomainOffset) + stfVersion, nil
}
