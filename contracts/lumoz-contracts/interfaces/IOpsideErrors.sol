// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

interface IOpsideErrors {
    /**
     * @dev Thrown when the caller is not the OpenRegistrar
     */
    error OnlyOpenRegistrar();

    /**
     * @dev Thrown when the caller is not the OpsideSlots
     */
    error OnlyOpsideSlots();

    /**
     * @dev Thrown when the caller is not the SlotAdapter
     */
    error OnlySlotAdapter();

    error OnlyManager();

    error OnlyZkEvmContract();
}