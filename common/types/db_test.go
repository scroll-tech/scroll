package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRollerProveStatus(t *testing.T) {
	tests := []struct {
		name string
		s    RollerProveStatus
		want string
	}{
		{
			"RollerAssigned",
			RollerAssigned,
			"RollerAssigned",
		},
		{
			"RollerProofValid",
			RollerProofValid,
			"RollerProofValid",
		},
		{
			"RollerProofInvalid",
			RollerProofInvalid,
			"RollerProofInvalid",
		},
		{
			"Bad Value",
			RollerProveStatus(999), // Invalid value.
			"Bad Value: 999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}

func TestProvingStatus(t *testing.T) {
	tests := []struct {
		name string
		s    ProvingStatus
		want string
	}{
		{
			"ProvingTaskUnassigned",
			ProvingTaskUnassigned,
			"unassigned",
		},
		{
			"ProvingTaskSkipped",
			ProvingTaskSkipped,
			"skipped",
		},
		{
			"ProvingTaskAssigned",
			ProvingTaskAssigned,
			"assigned",
		},
		{
			"ProvingTaskProved",
			ProvingTaskProved,
			"proved",
		},
		{
			"ProvingTaskVerified",
			ProvingTaskVerified,
			"verified",
		},
		{
			"ProvingTaskFailed",
			ProvingTaskFailed,
			"failed",
		},
		{
			"Undefined",
			ProvingStatus(999), // Invalid value.
			"Undefined (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}
