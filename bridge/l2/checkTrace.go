package l2

import (
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
)

//nolint:unused
func blockTraceIsValid(trace *types.BlockResult) bool {
	if trace == nil {
		log.Warn("block trace is empty")
		return false
	}
	flag := true
	for _, tx := range trace.ExecutionResults {
		flag = structLogResIsValid(tx.StructLogs) && flag
	}
	return flag
}

//nolint:unused
func structLogResIsValid(txLogs []*types.StructLogRes) bool {
	res := true
	for i := 0; i < len(txLogs); i++ {
		txLog := txLogs[i]
		flag := true
		switch vm.StringToOp(txLog.Op) {
		case vm.CALL, vm.CALLCODE:
			flag = codeIsValid(txLog, 2) && flag
			flag = stateIsValid(txLog, 2) && flag
		case vm.DELEGATECALL, vm.STATICCALL:
			flag = codeIsValid(txLog, 2) && flag
		case vm.CREATE, vm.CREATE2:
			flag = stateIsValid(txLog, 1) && flag
		case vm.SLOAD, vm.SSTORE, vm.SELFBALANCE:
			flag = stateIsValid(txLog, 1) && flag
		case vm.SELFDESTRUCT:
			flag = stateIsValid(txLog, 2) && flag
		case vm.EXTCODEHASH, vm.BALANCE:
			flag = stateIsValid(txLog, 1) && flag
		}
		res = res && flag
	}

	return res
}

//nolint:unused
func codeIsValid(txLog *types.StructLogRes, n int) bool {
	extraData := txLog.ExtraData
	if extraData == nil {
		log.Warn("extraData is empty", "pc", txLog.Pc, "opcode", txLog.Op)
		return false
	} else if len(extraData.CodeList) < n {
		log.Warn("code list is too short", "opcode", txLog.Op, "expect length", n, "actual length", len(extraData.CodeList))
		return false
	}
	return true
}

//nolint:unused
func stateIsValid(txLog *types.StructLogRes, n int) bool {
	extraData := txLog.ExtraData
	if extraData == nil {
		log.Warn("extraData is empty", "pc", txLog.Pc, "opcode", txLog.Op)
		return false
	} else if len(extraData.StateList) < n {
		log.Warn("stateList list is too short", "opcode", txLog.Op, "expect length", n, "actual length", len(extraData.StateList))
		return false
	}
	return true
}

// TraceHasUnsupportedOpcodes check if exist unsupported opcodes
func TraceHasUnsupportedOpcodes(opcodes map[string]struct{}, trace *types.BlockResult) bool {
	if trace == nil {
		return false
	}
	eg := errgroup.Group{}
	for _, res := range trace.ExecutionResults {
		res := res
		eg.Go(func() error {
			for _, lg := range res.StructLogs {
				if _, ok := opcodes[lg.Op]; ok {
					return fmt.Errorf("unsupported opcde: %s", lg.Op)
				}
			}
			return nil
		})
	}

	err := eg.Wait()
	return err != nil
}
