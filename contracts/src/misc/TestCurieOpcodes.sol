// SPDX-License-Identifier: MIT
pragma solidity =0.8.24;

contract TestCurieOpcodes {
    event TloadSuccess(uint256 value);
    event TstoreSuccess();
    event McopySuccess(bytes32 data);
    event BaseFeeSuccess(uint256 basefee);

    uint256 private constant TESTSLOT = 1234567890;

    function useTloadTstore(uint256 newValue) external {
        uint256 oldValue;
        uint256 loadedNewValue;
        assembly {
            oldValue := sload(TESTSLOT)
            sstore(TESTSLOT, newValue)
            loadedNewValue := sload(TESTSLOT)
        }
        emit TloadSuccess(oldValue);
        emit TstoreSuccess();
        emit TloadSuccess(loadedNewValue);
    }

    function useMcopy() external {
        bytes32 copiedData;
        assembly {
            mstore(0x20, 0x50)
            mcopy(0, 0x20, 0x20)
            copiedData := mload(0)
        }
        emit McopySuccess(copiedData);
    }

    function useBaseFee() external {
        emit BaseFeeSuccess(block.basefee);
    }
}
