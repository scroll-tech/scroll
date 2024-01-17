package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProverProveStatus(t *testing.T) {
	tests := []struct {
		name string
		s    ProverProveStatus
		want string
	}{
		{
			"ProverAssigned",
			ProverAssigned,
			"ProverAssigned",
		},
		{
			"ProverProofValid",
			ProverProofValid,
			"ProverProofValid",
		},
		{
			"ProverProofInvalid",
			ProverProofInvalid,
			"ProverProofInvalid",
		},
		{
			"Bad Value",
			ProverProveStatus(999), // Invalid value.
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
			"ProvingTaskAssigned",
			ProvingTaskAssigned,
			"assigned",
		},
		{
			"ProvingTaskProvedDEPRECATED",
			ProvingTaskProvedDEPRECATED,
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

func TestRollupStatus(t *testing.T) {
	tests := []struct {
		name string
		s    RollupStatus
		want string
	}{
		{
			"RollupUndefined",
			RollupUndefined,
			"Undefined (0)",
		},
		{
			"RollupPending",
			RollupPending,
			"RollupPending",
		},
		{
			"RollupCommitting",
			RollupCommitting,
			"RollupCommitting",
		},
		{
			"RollupCommitted",
			RollupCommitted,
			"RollupCommitted",
		},
		{
			"RollupFinalizing",
			RollupFinalizing,
			"RollupFinalizing",
		},
		{
			"RollupFinalized",
			RollupFinalized,
			"RollupFinalized",
		},
		{
			"RollupCommitFailed",
			RollupCommitFailed,
			"RollupCommitFailed",
		},
		{
			"RollupFinalizeFailed",
			RollupFinalizeFailed,
			"RollupFinalizeFailed",
		},
		{
			"Invalid Value",
			RollupStatus(999),
			"Undefined (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}

func TestSenderType(t *testing.T) {
	tests := []struct {
		name string
		t    SenderType
		want string
	}{
		{
			"SenderTypeUnknown",
			SenderTypeUnknown,
			"Unknown (0)",
		},
		{
			"SenderTypeCommitBatch",
			SenderTypeCommitBatch,
			"SenderTypeCommitBatch",
		},
		{
			"SenderTypeFinalizeBatch",
			SenderTypeFinalizeBatch,
			"SenderTypeFinalizeBatch",
		},
		{
			"SenderTypeL1GasOracle",
			SenderTypeL1GasOracle,
			"SenderTypeL1GasOracle",
		},
		{
			"SenderTypeL2GasOracle",
			SenderTypeL2GasOracle,
			"SenderTypeL2GasOracle",
		},
		{
			"Invalid Value",
			SenderType(999),
			"Unknown (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.t.String())
		})
	}
}

func TestTxStatus(t *testing.T) {
	tests := []struct {
		name string
		s    TxStatus
		want string
	}{
		{
			"TxStatusUnknown",
			TxStatusUnknown,
			"Unknown (0)",
		},
		{
			"TxStatusPending",
			TxStatusPending,
			"TxStatusPending",
		},
		{
			"TxStatusReplaced",
			TxStatusReplaced,
			"TxStatusReplaced",
		},
		{
			"TxStatusConfirmed",
			TxStatusConfirmed,
			"TxStatusConfirmed",
		},
		{
			"TxStatusConfirmedFailed",
			TxStatusConfirmedFailed,
			"TxStatusConfirmedFailed",
		},
		{
			"Invalid Value",
			TxStatus(999),
			"Unknown (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}

func TestGasOracleStatus(t *testing.T) {
	tests := []struct {
		name string
		s    GasOracleStatus
		want string
	}{
		{
			"GasOracleUndefined",
			GasOracleUndefined,
			"GasOracleUndefined",
		},
		{
			"GasOraclePending",
			GasOraclePending,
			"GasOraclePending",
		},
		{
			"GasOracleImporting",
			GasOracleImporting,
			"GasOracleImporting",
		},
		{
			"GasOracleImported",
			GasOracleImported,
			"GasOracleImported",
		},
		{
			"GasOracleImportedFailed",
			GasOracleImportedFailed,
			"GasOracleImportedFailed",
		},
		{
			"Invalid Value",
			GasOracleStatus(999),
			"Undefined (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.s.String())
		})
	}
}

func TestProverTaskFailureType(t *testing.T) {
	tests := []struct {
		name string
		r    ProverTaskFailureType
		want string
	}{
		{
			"ProverTaskFailureTypeUndefined",
			ProverTaskFailureTypeUndefined,
			"prover task failure undefined",
		},
		{
			"ProverTaskFailureTypeTimeout",
			ProverTaskFailureTypeTimeout,
			"prover task failure timeout",
		},
		{
			"ProverTaskFailureTypeSubmitStatusNotOk",
			ProverTaskFailureTypeSubmitStatusNotOk,
			"prover task failure validated submit proof status not ok",
		},
		{
			"ProverTaskFailureTypeVerifiedFailed",
			ProverTaskFailureTypeVerifiedFailed,
			"prover task failure verified failed",
		},
		{
			"ProverTaskFailureTypeServerError",
			ProverTaskFailureTypeServerError,
			"prover task failure server exception",
		},
		{
			"Invalid Value",
			ProverTaskFailureType(999),
			"illegal prover task failure type (999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.r.String())
		})
	}
}
