"""
Derive Galileo Gas Parameters from on-chain data

This script derives the L1 fee parameters for Galileo upgrade:
- commit_scalar
- blob_scalar
- penalty_multiplier

Using a two-stage algebraic approach:
1. Stage 1: Set penalty_multiplier based on P95 transaction size
2. Stage 2: Solve commit_scalar and blob_scalar using simple algebraic division
"""

from web3 import Web3
import pandas as pd
import numpy as np
import pickle
import zstandard as zstd
import rlp
import math
import requests
import time
import os
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed
from multiprocessing import Pool, cpu_count

# ============================================================================
# CONSTANTS
# ============================================================================

# Read RPC endpoints from environment variables
mainnet_url = os.getenv('MAINNET_URL', '')
scroll_url = os.getenv('SCROLL_URL', 'https://rpc.scroll.io')
beacon_url = os.getenv('BEACON_URL', '')

# Validate that environment variables are set
if not mainnet_url:
    raise ValueError("MAINNET_URL environment variable is not set. Please set it to your Ethereum mainnet RPC endpoint.")
if not scroll_url:
    raise ValueError("SCROLL_URL environment variable is not set. Please set it to your Scroll RPC endpoint.")
if not beacon_url:
    raise ValueError("BEACON_URL environment variable is not set. Please set it to your Ethereum beacon chain RPC endpoint.")

# EtherFi contract addresses (lowercase for comparison)
ETHERFI_SPEND_ADDRESS = "0x7ca0b75e67e33c0014325b739a8d019c4fe445f0"
ETHERFI_SWAP_ADDRESS = "0x4deaa5f2e1cd1a792304d1649edfa35d565f9346"

# ============================================================================
# ABIs
# ============================================================================

rollup_abi = [
    {
        "anonymous": False,
        "inputs": [
            {"indexed": True, "internalType": "uint256", "name": "batchIndex", "type": "uint256"},
            {"indexed": True, "internalType": "bytes32", "name": "batchHash", "type": "bytes32"}
        ],
        "name": "CommitBatch",
        "type": "event"
    },
    {
        "anonymous": False,
        "inputs": [
            {"indexed": True, "internalType": "uint256", "name": "batchIndex", "type": "uint256"},
            {"indexed": True, "internalType": "bytes32", "name": "batchHash", "type": "bytes32"},
            {"indexed": False, "internalType": "bytes32", "name": "stateRoot", "type": "bytes32"},
            {"indexed": False, "internalType": "bytes32", "name": "withdrawRoot", "type": "bytes32"}
        ],
        "name": "FinalizeBatch",
        "type": "event"
    },
]

fee_oracle_abi = [
    {
        "type": "event",
        "name": "L1BaseFeeUpdated",
        "inputs": [{"name": "l1BaseFee","type": "uint256","indexed": False,"internalType": "uint256"}],
        "anonymous": False
    },
    {
        "type": "event",
        "name": "L1BlobBaseFeeUpdated",
        "inputs": [{"name": "l1BlobBaseFee","type": "uint256","indexed": False,"internalType": "uint256"}],
        "anonymous": False
    },
    {
        "type": "function",
        "name": "l1BaseFee",
        "inputs": [],
        "outputs": [{"name": "", "type": "uint256", "internalType": "uint256"}],
        "stateMutability": "view"
    },
    {
        "type": "function",
        "name": "l1BlobBaseFee",
        "inputs": [],
        "outputs": [{"name": "", "type": "uint256", "internalType": "uint256"}],
        "stateMutability": "view"
    },
    {
        "type": "function",
        "name": "commitScalar",
        "inputs": [],
        "outputs": [{"name": "", "type": "uint256", "internalType": "uint256"}],
        "stateMutability": "view"
    },
    {
        "type": "function",
        "name": "blobScalar",
        "inputs": [],
        "outputs": [{"name": "", "type": "uint256", "internalType": "uint256"}],
        "stateMutability": "view"
    },
    {
        "type": "function",
        "name": "penaltyFactor",
        "inputs": [],
        "outputs": [{"name": "", "type": "uint256", "internalType": "uint256"}],
        "stateMutability": "view"
    },
    {
        "type": "function",
        "name": "isGalileo",
        "inputs": [],
        "outputs": [{"name": "", "type": "bool", "internalType": "bool"}],
        "stateMutability": "view"
    },
]

# ============================================================================
# WEB3 SETUP
# ============================================================================

w3 = Web3(Web3.HTTPProvider(mainnet_url))
scroll_w3 = Web3(Web3.HTTPProvider(scroll_url))

rollup_contract_address = Web3.to_checksum_address("0xa13BAF47339d63B743e7Da8741db5456DAc1E556")
rollup_contract = w3.eth.contract(address=rollup_contract_address, abi=rollup_abi)

l1_fee_oracle_contract_address = Web3.to_checksum_address("0x5300000000000000000000000000000000000002")
l1_fee_oracle_contract = scroll_w3.eth.contract(address=l1_fee_oracle_contract_address, abi=fee_oracle_abi)


def read_current_gas_parameters():
    """Read and display current on-chain Galileo gas parameters from L1GasPriceOracle"""
    print("=" * 60)
    print("CURRENT ON-CHAIN GAS PARAMETERS (Galileo)")
    print("=" * 60)

    is_galileo = l1_fee_oracle_contract.functions.isGalileo().call()
    l1_base_fee = l1_fee_oracle_contract.functions.l1BaseFee().call()
    l1_blob_base_fee = l1_fee_oracle_contract.functions.l1BlobBaseFee().call()
    commit_scalar_raw = l1_fee_oracle_contract.functions.commitScalar().call()
    blob_scalar_raw = l1_fee_oracle_contract.functions.blobScalar().call()
    penalty_factor = l1_fee_oracle_contract.functions.penaltyFactor().call()

    commit_scalar = commit_scalar_raw / 1e9
    blob_scalar = blob_scalar_raw / 1e9

    print(f"  isGalileo:      {is_galileo}")
    print(f"  l1BaseFee:      {l1_base_fee} wei ({l1_base_fee / 1e9:.2f} gwei)")
    print(f"  l1BlobBaseFee:  {l1_blob_base_fee} wei ({l1_blob_base_fee / 1e9:.2f} gwei)")
    print(f"  commitScalar:   {commit_scalar_raw} (decoded: {commit_scalar:.4f})")
    print(f"  blobScalar:     {blob_scalar_raw} (decoded: {blob_scalar:.4f})")
    print(f"  penaltyFactor:  {penalty_factor}")
    print("=" * 60)

    return {
        'l1_base_fee': l1_base_fee,
        'l1_blob_base_fee': l1_blob_base_fee,
        'commit_scalar_raw': commit_scalar_raw,
        'blob_scalar_raw': blob_scalar_raw,
        'commit_scalar': commit_scalar,
        'blob_scalar': blob_scalar,
        'penalty_factor': penalty_factor,
    }


# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

def reverse_make_canonical(input_bytes, n=31):
    """Reverse effect of make_canonical"""
    output_bytes = bytearray()
    for i in range(0, len(input_bytes)):
        if i % (n+1) != 0:
            output_bytes.extend(input_bytes[i:i+1])
    return output_bytes


def kzg_to_versioned_hash(kzg_commitment):
    """Given kzg_commitment returns versioned_hash of version 0x01"""
    from hashlib import sha256
    return "0x01"+sha256(bytes.fromhex(kzg_commitment[2:])).hexdigest()[2:]


def latest_finalized_event_block(width=5):
    """Find the latest L1 block with a FinalizeBatch event"""
    finalized_l1_head = -1

    to_block = w3.eth.block_number
    from_block = to_block - width

    while finalized_l1_head == -1:
        event_filter = rollup_contract.events.FinalizeBatch.create_filter(fromBlock=from_block, toBlock=to_block)
        events = event_filter.get_all_entries()

        if len(events) > 0:
            finalized_l1_head = events[-1]['blockNumber']
            break

        to_block = from_block - 1
        from_block = from_block - width - 1

    return finalized_l1_head


def beacon_head(l1_head, mainnet_beacon_url):
    """Get beacon chain slot number for given L1 block"""
    latest_block_number = w3.eth.block_number
    url = f"{mainnet_beacon_url}/eth/v1/beacon/headers/head"
    headers = {'accept': 'application/json'}
    response = requests.get(url, headers=headers)
    return int(response.json()['data']['header']['message']['slot']) - (latest_block_number - l1_head)


def to_bytes_helper(value):
    """Convert various types to bytes"""
    if isinstance(value, bytes):
        return value
    elif isinstance(value, int):
        # Handle integer values (convert to bytes)
        if value == 0:
            return b''
        return value.to_bytes((value.bit_length() + 7) // 8, 'big')
    elif hasattr(value, 'hex'):
        # Handle HexBytes and similar objects
        try:
            hex_str = value.hex()
            # Check if hex_str contains only valid hex characters
            hex_clean = hex_str[2:] if hex_str.startswith('0x') else hex_str
            # Validate hex string before conversion
            if all(c in '0123456789abcdefABCDEF' for c in hex_clean):
                # Ensure hex string has even length (pad with leading zero if odd)
                if len(hex_clean) % 2 == 1:
                    hex_clean = '0' + hex_clean
                return bytes.fromhex(hex_clean)
            else:
                # Invalid hex, try direct bytes conversion
                return bytes(value)
        except (ValueError, AttributeError):
            # If hex() method exists but produces invalid output, try direct bytes conversion
            return bytes(value)
    elif isinstance(value, str):
        hex_clean = value[2:] if value.startswith('0x') else value
        # Validate hex string before conversion
        if all(c in '0123456789abcdefABCDEF' for c in hex_clean):
            # Ensure hex string has even length (pad with leading zero if odd)
            if len(hex_clean) % 2 == 1:
                hex_clean = '0' + hex_clean
            return bytes.fromhex(hex_clean)
        else:
            raise ValueError(f"Invalid hex string: {value}")
    else:
        try:
            return bytes(value)
        except TypeError:
            # Last resort: convert to int first then to bytes
            int_val = int(value)
            if int_val == 0:
                return b''
            return int_val.to_bytes((int_val.bit_length() + 7) // 8, 'big')


def get_raw_transaction_from_structured(tx):
    """
    Convert structured transaction to raw transaction bytes
    Supports: Legacy (type 0), EIP-2930 (type 1), EIP-1559 (type 2), L1 message (type 126), EIP-7702 (type 4)
    """
    tx_type = tx.get('type')

    if tx_type == 126:
        queue_index = tx['queueIndex']
        if hasattr(queue_index, 'hex'):
            queue_index = queue_index.hex()

        gas_val = tx['gas']
        value_val = tx['value']

        to_addr = tx['to']
        if hasattr(to_addr, 'hex'):
            to_addr = to_addr.hex()

        sender_addr = tx['sender']
        if hasattr(sender_addr, 'hex'):
            sender_addr = sender_addr.hex()

        elements = [
            bytes.fromhex(queue_index[2:].rjust(6, "0")),
            gas_val.to_bytes(max(1, math.ceil(gas_val.bit_length() / 8)), 'big'),
            bytes.fromhex(to_addr[2:]),
            value_val.to_bytes(max(1, math.ceil(value_val.bit_length() / 8)), 'big') if value_val > 0 else b'',
            to_bytes_helper(tx['input']),
            bytes.fromhex(sender_addr[2:])
        ]
        return b'~'+rlp.encode(elements)

    elif tx_type == 2:  # EIP-1559 Transaction
        to_address_bytes = bytes.fromhex(tx['to'][2:]) if tx.get('to') else b''

        access_list_rlp = []
        for item in tx.get('accessList', []):
            storage_keys = [to_bytes_helper(sk) for sk in item['storageKeys']]
            access_list_rlp.append([bytes.fromhex(item['address'][2:]), storage_keys])

        elements = [
            tx['chainId'],
            tx['nonce'],
            tx['maxPriorityFeePerGas'],
            tx['maxFeePerGas'],
            tx['gas'],
            to_address_bytes,
            tx['value'],
            to_bytes_helper(tx['input']),
            access_list_rlp,
            tx['yParity'],
            to_bytes_helper(tx['r']),
            to_bytes_helper(tx['s'])
        ]
        return b'\x02' + rlp.encode(elements)

    elif tx_type == 1:  # EIP-2930 Access List Transaction
        to_address_bytes = bytes.fromhex(tx['to'][2:]) if tx.get('to') else b''

        access_list_rlp = []
        for item in tx.get('accessList', []):
            storage_keys = [to_bytes_helper(sk) for sk in item['storageKeys']]
            access_list_rlp.append([bytes.fromhex(item['address'][2:]), storage_keys])

        elements = [
            tx['chainId'],
            tx['nonce'],
            tx['gasPrice'],
            tx['gas'],
            to_address_bytes,
            tx['value'],
            to_bytes_helper(tx['input']),
            access_list_rlp,
            tx['yParity'],
            to_bytes_helper(tx['r']),
            to_bytes_helper(tx['s'])
        ]
        return b'\x01' + rlp.encode(elements)

    elif tx_type == 0 or tx_type is None:  # Legacy Transaction
        to_address_bytes = bytes.fromhex(tx['to'][2:]) if tx.get('to') else b''

        elements = [
            tx['nonce'],
            tx['gasPrice'],
            tx['gas'],
            to_address_bytes,
            tx['value'],
            to_bytes_helper(tx['input']),
            tx['v'],
            to_bytes_helper(tx['r']),
            to_bytes_helper(tx['s'])
        ]
        return rlp.encode(elements)

    elif tx_type == 4:  # EIP-7702 Transaction
        to_address_bytes = bytes.fromhex(tx['to'][2:]) if tx.get('to') else b''

        access_list_rlp = []
        for item in tx.get('accessList', []):
            storage_keys = [to_bytes_helper(sk) for sk in item['storageKeys']]
            access_list_rlp.append([bytes.fromhex(item['address'][2:]), storage_keys])

        authorization_list_rlp = []
        for auth in tx.get('authorizationList', []):
            authorization_list_rlp.append([
                auth['chainId'],
                bytes.fromhex(auth['address'][2:]),
                auth['nonce'],
                auth['yParity'],
                to_bytes_helper(auth['r']),
                to_bytes_helper(auth['s'])
            ])

        elements = [
            tx['chainId'],
            tx['nonce'],
            tx['maxPriorityFeePerGas'],
            tx['maxFeePerGas'],
            tx['gas'],
            to_address_bytes,
            tx['value'],
            to_bytes_helper(tx['input']),
            access_list_rlp,
            authorization_list_rlp,
            tx['yParity'],
            to_bytes_helper(tx['r']),
            to_bytes_helper(tx['s'])
        ]
        return b'\x04' + rlp.encode(elements)

    else:
        raise ValueError(f"Unsupported transaction type: {tx_type}")


def get_transaction_byte_length(tx):
    """Get transaction size in bytes"""
    raw_tx = get_raw_transaction_from_structured(tx)
    return len(raw_tx)


def compress_transaction(tx):
    """
    Compress transaction using zstd level 3
    Returns compressed size (with min to handle compression failure)
    """
    raw_tx = get_raw_transaction_from_structured(tx)
    compressed = zstd.compress(raw_tx, level=3)
    # Handle compression failure: use min of compressed and original
    return min(len(compressed), len(raw_tx))


def process_transaction_worker(tx_dict):
    """
    Worker function for parallel transaction processing
    Takes a dict representation of transaction and returns sizes

    Args:
        tx_dict: Dictionary representation of transaction

    Returns:
        (tx_hash, size, compressed_size)
    """
    size = get_transaction_byte_length(tx_dict)
    compressed_size = compress_transaction(tx_dict)
    return (tx_dict['hash'].hex() if hasattr(tx_dict['hash'], 'hex') else tx_dict['hash'],
            size,
            compressed_size)


# ============================================================================
# BLOB AND BATCH PARSING
# ============================================================================

# Global cache for blob data
indexed_blob = {}


def get_blob_data_v2(blob_hash, block_number, mainnet_beacon_url, cur_slot, cur_block):
    """
    Fetch blob data from beacon chain
    Returns: blob data, updated cur_slot, updated cur_block
    """
    global indexed_blob

    if blob_hash in indexed_blob:
        return indexed_blob[blob_hash], cur_slot, cur_block

    cur_slot = cur_slot - (cur_block - block_number) + 1
    cur_block = block_number

    found_the_blob = False
    while not found_the_blob:
        cur_slot -= 1
        url = f"{mainnet_beacon_url}/eth/v1/beacon/blob_sidecars/{cur_slot}"

        headers = {'accept': 'application/json'}
        try:
            response = requests.get(url, headers=headers, timeout=5)
        except Exception as e:
            print(f"[warn] request error at slot {cur_slot}: {e}")
            continue

        if response.status_code != 200:
            print(f"[warn] non-200 from beacon at slot {cur_slot}: {response.status_code}")
            continue

        if 'data' in response.json().keys():
            for blob in response.json()['data']:
                hash = kzg_to_versioned_hash(blob['kzg_commitment'])
                if blob_hash == hash:
                    found_the_blob = True
                indexed_blob[hash] = blob['blob']

    return indexed_blob[blob_hash], cur_slot, cur_block


def parse_batch(blob_hash, block_number, mainnet_beacon_url, cur_slot, cur_block):
    """
    Parse batch from blob data to get L2 block range
    Returns: (initial_L2_block_number, num_blocks, cur_slot, cur_block)
    """
    ZSTD_MAGIC = b'\x28\xb5\x2f\xfd'

    batch_data, cur_slot, cur_block = get_blob_data_v2(blob_hash, block_number, mainnet_beacon_url, cur_slot, cur_block)
    batch_data = reverse_make_canonical(bytearray.fromhex(batch_data[2:]), 31)

    version = int(batch_data[0])
    payload_N = int.from_bytes(batch_data[1:4], 'big')
    flag = batch_data[4]
    payload = batch_data[5:5+payload_N]
    if flag == 1:
        payload = zstd.decompress(ZSTD_MAGIC+payload)
    initial_L2_block_number = int.from_bytes(payload[64:64+8], 'big')
    num_blocks = int.from_bytes(payload[72:72+2], 'big')

    return (initial_L2_block_number, num_blocks, cur_slot, cur_block)


# ============================================================================
# DATA COLLECTION FUNCTIONS
# ============================================================================

def collect_batch_data(n_batches=30, width=5, start_time=None):
    """
    Collect batch data from L1 (commits and finalizations)

    Strategy: First find finalized batches, then collect their commit data

    Args:
        n_batches: Number of batches to collect
        width: Block range width for event filtering
        start_time: Script start time for elapsed time calculation

    Returns:
        batch_df: DataFrame with columns:
            - index (batch_index)
            - versioned_hash
            - initial_L2_block_number
            - num_blocks
            - commit_cost
            - finalize_cost
            - blob_cost
    """
    print("=" * 60)
    print(f"COLLECTING BATCH DATA ({n_batches} batches)")
    print("=" * 60)

    # Get L1 head
    mainnet_beacon_url = beacon_url
    l1_head = latest_finalized_event_block(width)
    beacon_head_slot = beacon_head(l1_head, mainnet_beacon_url)

    print(f"L1 HEAD: {l1_head}")
    print(f"Beacon HEAD: {beacon_head_slot}")

    # Step 1: Collect FinalizeBatch events until we have enough batches
    print(f"\nStep 1: Collecting FinalizeBatch events...")
    finalize_events = []  # List of (batch_index, finalize_cost)
    to_block = l1_head
    from_block = to_block - width
    total_batches_covered = 0
    prev_event_count = 0

    # Keep scanning until we have enough batches
    # We need at least one extra event to properly calculate the range
    while total_batches_covered < n_batches or len(finalize_events) < 2:
        event_filter = rollup_contract.events.FinalizeBatch.create_filter(fromBlock=from_block, toBlock=to_block)
        events = event_filter.get_all_entries()

        for event in events:
            batch_index = event['args']['batchIndex']
            receipt = w3.eth.get_transaction_receipt(event.transactionHash)
            finalize_cost = receipt['gasUsed'] * receipt['effectiveGasPrice']
            finalize_events.append((batch_index, finalize_cost))

        # Calculate how many batches we have covered so far
        if len(finalize_events) >= 2:
            finalize_events.sort(key=lambda x: x[0], reverse=True)
            latest_batch = finalize_events[0][0]
            # To properly calculate coverage, we need the range from the second-to-last event
            # to the latest event, because we'll select N batches ending at latest_batch
            if len(finalize_events) >= 2:
                second_oldest = finalize_events[-2][0]
                # The batches we can properly account for are from second_oldest+1 to latest_batch
                total_batches_covered = latest_batch - second_oldest

        # Only print when we find new events
        if len(finalize_events) > prev_event_count:
            print(f"  Found {len(finalize_events)} FinalizeBatch events, covering ~{total_batches_covered} batches")
            prev_event_count = len(finalize_events)

        to_block = from_block - 1
        from_block = from_block - width - 1

        if from_block < 0:
            break

    if len(finalize_events) == 0:
        raise RuntimeError("Could not find any FinalizeBatch events")

    # Sort events by batch index (descending - newest first)
    finalize_events.sort(key=lambda x: x[0], reverse=True)
    print(f"\nFinalizeBatch events found:")
    for batch_idx, cost in finalize_events:
        print(f"  Batch {batch_idx}: {cost:,} wei")

    # Calculate which batches each FinalizeBatch event covers
    # and compute per-batch finalize costs
    batch_finalize_costs = {}  # batch_index -> finalize_cost
    all_finalized_batches = set()

    for i, (batch_index, finalize_cost) in enumerate(finalize_events):
        # Determine the range this FinalizeBatch covers
        if i == len(finalize_events) - 1:
            # This is the oldest event - we don't know where it starts
            # For now, assume it only finalizes this single batch
            start_batch = batch_index
        else:
            # This event finalizes from (next_older_batch + 1) to current_batch
            next_older_batch = finalize_events[i + 1][0]
            start_batch = next_older_batch + 1

        end_batch = batch_index
        num_batches = end_batch - start_batch + 1
        per_batch_cost = finalize_cost / num_batches

        print(f"\n  FinalizeBatch({batch_index}) covers batches {start_batch}-{end_batch} ({num_batches} batches)")
        print(f"    Total cost: {finalize_cost:,} wei")
        print(f"    Per-batch cost: {per_batch_cost:,.2f} wei")

        # Record the finalize cost for each batch in this range
        for b in range(start_batch, end_batch + 1):
            batch_finalize_costs[b] = per_batch_cost
            all_finalized_batches.add(b)

    # Select the latest N batches
    latest_batch = max(all_finalized_batches)
    finalized_batch_indices = list(range(latest_batch - n_batches + 1, latest_batch + 1))

    print(f"\n  Selected {len(finalized_batch_indices)} consecutive batches: {finalized_batch_indices[0]} - {finalized_batch_indices[-1]}")

    # Verify all selected batches have finalize costs
    missing_batches = [b for b in finalized_batch_indices if b not in batch_finalize_costs]
    if missing_batches:
        print(f"  Warning: {len(missing_batches)} batches missing finalize data: {missing_batches[:10]}...")
        # Need to scan further back
        print(f"  Scanning further back to find more FinalizeBatch events...")
        # Continue scanning...
        while missing_batches and from_block >= 0:
            to_block = from_block - 1
            from_block = from_block - width - 1

            event_filter = rollup_contract.events.FinalizeBatch.create_filter(fromBlock=from_block, toBlock=to_block)
            events = event_filter.get_all_entries()

            for event in events:
                batch_index = event['args']['batchIndex']
                receipt = w3.eth.get_transaction_receipt(event.transactionHash)
                finalize_cost = receipt['gasUsed'] * receipt['effectiveGasPrice']
                finalize_events.append((batch_index, finalize_cost))

            if len(events) > 0:
                # Recalculate with new events
                finalize_events.sort(key=lambda x: x[0], reverse=True)
                batch_finalize_costs = {}

                for i, (batch_index, finalize_cost) in enumerate(finalize_events):
                    if i == len(finalize_events) - 1:
                        start_batch = batch_index
                    else:
                        next_older_batch = finalize_events[i + 1][0]
                        start_batch = next_older_batch + 1

                    end_batch = batch_index
                    num_batches = end_batch - start_batch + 1
                    per_batch_cost = finalize_cost / num_batches

                    for b in range(start_batch, end_batch + 1):
                        batch_finalize_costs[b] = per_batch_cost

                missing_batches = [b for b in finalized_batch_indices if b not in batch_finalize_costs]
                print(f"    Still missing {len(missing_batches)} batches...")

                if not missing_batches:
                    break

    # Step 2: Now collect commit data for these specific batches
    print(f"\nStep 2: Collecting commit data for finalized batches...")
    batch_data_dict = {}  # batch_index -> [versioned_hash, initial_L2_block_number, num_blocks, commit_cost, blob_cost]
    to_block = l1_head
    from_block = to_block - width
    cur_slot = beacon_head_slot
    cur_block = l1_head
    prev_batch_count = 0

    # We need to find commits for all batches in finalized_batch_indices
    target_batches = set(finalized_batch_indices)
    last_tx_hash = None
    blob_index = 0

    while len(batch_data_dict) < n_batches:
        event_filter = rollup_contract.events.CommitBatch.create_filter(fromBlock=from_block, toBlock=to_block)
        events = event_filter.get_all_entries()

        for event in events:
            batch_index = event['args']['batchIndex']
            tx_hash = event.transactionHash

            # Track blob index for multiple batches in same transaction
            # This must be done BEFORE filtering, so we count all batches in the tx
            if last_tx_hash == tx_hash:
                blob_index += 1
            else:
                last_tx_hash = tx_hash
                blob_index = 0

            # Only process if this batch is in our finalized list
            if batch_index not in target_batches:
                continue

            # Skip if we already have this batch
            if batch_index in batch_data_dict:
                continue

            tx = w3.eth.get_transaction(event.transactionHash)
            receipt = w3.eth.get_transaction_receipt(event.transactionHash)

            block_number = tx.blockNumber
            num_batches_in_tx = len(tx.blobVersionedHashes)
            blob_hash = tx.blobVersionedHashes[blob_index].hex()

            # Calculate commit cost (execution gas only, amortized across all batches in this tx)
            commit_cost = (receipt['gasUsed'] * receipt['effectiveGasPrice']) / num_batches_in_tx

            # Parse batch to get L2 block range
            initial_L2_block_number, num_blocks, cur_slot, cur_block = parse_batch(
                blob_hash, block_number, mainnet_beacon_url, cur_slot, cur_block
            )

            # Calculate blob cost (amortized across all batches in this tx)
            blob_cost = (receipt['blobGasPrice'] * receipt['blobGasUsed']) / num_batches_in_tx

            batch_data_dict[batch_index] = [blob_hash, initial_L2_block_number, num_blocks,
                                              commit_cost, blob_cost]

        # Only print when count changes
        if len(batch_data_dict) > prev_batch_count:
            print(f"  Collected commit data for {len(batch_data_dict)}/{n_batches} batches")
            prev_batch_count = len(batch_data_dict)

        to_block = from_block - 1
        from_block = from_block - width - 1

        # Stop if we've searched far enough
        if from_block < 0:
            break

    # Step 3: Combine commit and finalize data
    print(f"\nStep 3: Combining commit and finalize data...")
    batch_list = []
    for batch_index in sorted(batch_data_dict.keys(), reverse=True):
        if batch_index not in batch_finalize_costs:
            print(f"  Warning: Batch {batch_index} has commit but no finalize data (skipping)")
            continue

        blob_hash, initial_L2_block_number, num_blocks, commit_cost, blob_cost = batch_data_dict[batch_index]
        finalize_cost = batch_finalize_costs[batch_index]

        batch_list.append([
            batch_index,
            blob_hash,
            initial_L2_block_number,
            num_blocks,
            commit_cost,
            finalize_cost,
            blob_cost
        ])

    # Create DataFrame
    column_names = ['index', 'versioned_hash', 'initial_L2_block_number', 'num_blocks',
                    'commit_cost', 'finalize_cost', 'blob_cost']
    batch_df = pd.DataFrame(batch_list, columns=column_names).astype(object)
    batch_df = batch_df.set_index('index').sort_index()

    print(f"  Successfully collected {len(batch_df)} complete batches")
    print(f"  Batch range: {batch_df.index[0]} - {batch_df.index[-1]}")
    if start_time:
        print(f"\n✓ Batch data collection completed in {time.time() - start_time:.1f}s")

    return batch_df


def collect_transaction_data(batch_df, start_time=None):
    """
    Collect L2 transaction data for all batches

    Args:
        batch_df: DataFrame with batch information
        start_time: Script start time for elapsed time calculation

    Returns:
        tx_df: DataFrame with columns:
            - hash
            - block_number
            - batch_index
            - size
            - compressed_tx_size
            - l1_base_fee
            - l1_blob_base_fee
            - L1_fee
            - gas_price
            - gas_used
            - base_fee_per_gas
            - to
    """
    print("=" * 60)
    print("COLLECTING TRANSACTION DATA")
    print("=" * 60)

    # Calculate L2 block range (batches may not be in order)
    first_block = int(batch_df['initial_L2_block_number'].min())
    last_block = int((batch_df['initial_L2_block_number'] + batch_df['num_blocks']).max())

    print(f"L2 block range: {first_block} - {last_block}")
    print(f"\n  Fetching L2 blocks...")

    # Fetch all blocks in parallel
    indexed_block = {}
    total_blocks = last_block - first_block

    def fetch_block(block_number):
        """Fetch a single block with full transactions, with retry on rate limit"""
        max_retries = 5
        retry_delay = 2  # seconds

        for attempt in range(max_retries):
            try:
                return block_number, scroll_w3.eth.get_block(block_number, full_transactions=True)
            except Exception as e:
                if '429' in str(e) or 'Too Many Requests' in str(e):
                    wait_time = retry_delay * (attempt + 1)
                    print(f"    [WARN] Rate limit hit (429) for block {block_number}, retrying in {wait_time}s (attempt {attempt + 1}/{max_retries})")
                    if attempt < max_retries - 1:
                        time.sleep(wait_time)  # Exponential backoff
                        continue
                    else:
                        print(f"    [ERROR] Max retries reached for block {block_number}")
                raise

    # Use ThreadPoolExecutor for parallel RPC calls
    # With paid endpoint, can use more workers
    max_workers = 20  # Higher limit for paid RPC endpoint
    print(f"    Starting parallel fetch with {max_workers} workers...")

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(fetch_block, bn): bn for bn in range(first_block, last_block)}

        completed = 0
        start_fetch = time.time()
        for future in as_completed(futures):
            block_number, block = future.result()
            indexed_block[block_number] = block
            completed += 1

            # More frequent progress updates (every 50 blocks or major milestones)
            if completed % 50 == 0 or completed == total_blocks:
                elapsed = time.time() - start_fetch
                rate = completed / elapsed if elapsed > 0 else 0
                eta = (total_blocks - completed) / rate if rate > 0 else 0
                print(f"    Fetched {completed}/{total_blocks} blocks ({completed*100//total_blocks}%, {rate:.1f} blocks/s, ETA: {eta:.0f}s)")

    fetch_blocks_elapsed = time.time() - start_fetch
    print(f"\n  ✓ Block fetching completed in {fetch_blocks_elapsed:.1f}s")

    # Get initial L1 fee oracle values
    print(f"\n  Recording L1 fee oracle updates...")
    l1_base_fee = 0
    l1_blob_base_fee = 0
    cur_block_num = first_block

    while l1_base_fee == 0:
        block = scroll_w3.eth.get_block(cur_block_num, full_transactions=True)
        for tx in reversed(block['transactions']):
            if tx.to == '0x5300000000000000000000000000000000000002' and tx.input.hex()[:10] == '0x39455d3a':
                l1_base_fee = int.from_bytes(bytes.fromhex(tx.input.hex()[2:])[-64:-32], 'big')
                l1_blob_base_fee = int.from_bytes(bytes.fromhex(tx.input.hex()[2:])[-32:], 'big')
        cur_block_num -= 1

    # Process transactions in parallel
    print(f"\n  Processing transactions in parallel...")

    # Collect all transactions with their metadata first
    all_txs = []
    for block_number in range(first_block, last_block):
        block = indexed_block[block_number]
        for tx in block['transactions']:
            # Convert AttributeDict to regular dict for pickling
            tx_dict = dict(tx)
            all_txs.append((block_number, tx_dict))

    print(f"  Total transactions to process: {len(all_txs)}")

    # Parallel processing of transaction sizes
    tx_sizes = {}  # tx_hash -> (size, compressed_size)
    num_workers = min(cpu_count(), 8)  # Limit to 8 workers to avoid overhead

    start_process_tx = time.time()
    with Pool(processes=num_workers) as pool:
        # Process in chunks for progress reporting
        chunk_size = 500
        processed = 0

        for i in range(0, len(all_txs), chunk_size):
            chunk = [tx_dict for _, tx_dict in all_txs[i:i+chunk_size]]
            results = pool.map(process_transaction_worker, chunk)

            for tx_hash, size, compressed_size in results:
                tx_sizes[tx_hash] = (size, compressed_size)

            processed += len(chunk)
            print(f"    Processed {processed}/{len(all_txs)} transactions...")

    process_tx_elapsed = time.time() - start_process_tx
    print(f"\n  ✓ Transaction processing completed in {process_tx_elapsed:.1f}s")

    # Fetch transaction receipts in parallel to get gas_used and L1_fee
    print(f"\n  Fetching transaction receipts...")
    tx_receipts = {}  # tx_hash -> receipt

    def fetch_receipt(tx_hash_str):
        """Fetch a single transaction receipt with retry"""
        max_retries = 5
        retry_delay = 2

        for attempt in range(max_retries):
            try:
                return tx_hash_str, scroll_w3.eth.get_transaction_receipt(tx_hash_str)
            except Exception as e:
                if '429' in str(e) or 'Too Many Requests' in str(e):
                    wait_time = retry_delay * (attempt + 1)
                    if attempt < max_retries - 1:
                        time.sleep(wait_time)
                        continue
                raise

    start_fetch_receipts = time.time()
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        tx_hashes = [tx_dict['hash'].hex() if hasattr(tx_dict['hash'], 'hex') else tx_dict['hash']
                     for _, tx_dict in all_txs]
        futures = {executor.submit(fetch_receipt, tx_hash): tx_hash for tx_hash in tx_hashes}

        completed = 0
        for future in as_completed(futures):
            tx_hash, receipt = future.result()
            tx_receipts[tx_hash] = receipt
            completed += 1

            if completed % 500 == 0 or completed == len(tx_hashes):
                print(f"    Fetched {completed}/{len(tx_hashes)} receipts ({completed*100//len(tx_hashes)}%)")

    fetch_receipts_elapsed = time.time() - start_fetch_receipts
    print(f"\n  ✓ Receipt fetching completed in {fetch_receipts_elapsed:.1f}s")

    # Build rows with oracle fee tracking
    print(f"\n  Building transaction dataframe...")
    rows = []
    for block_number, tx_dict in all_txs:
        tx_hash = tx_dict['hash'].hex() if hasattr(tx_dict['hash'], 'hex') else tx_dict['hash']
        size, compressed_tx_size = tx_sizes[tx_hash]

        # Get receipt for gas_used and L1_fee
        receipt = tx_receipts.get(tx_hash)
        gas_used = receipt['gasUsed'] if receipt else 0
        L1_fee = receipt.get('l1Fee', 0) if receipt else 0  # L1 fee from receipt (Scroll-specific field)

        # Get gas_price from transaction
        gas_price = tx_dict.get('gasPrice', 0)
        if gas_price == 0 and 'maxFeePerGas' in tx_dict:
            # For EIP-1559 transactions, use maxFeePerGas if gasPrice not available
            gas_price = tx_dict.get('maxFeePerGas', 0)

        # Get base_fee_per_gas from block
        block = indexed_block[block_number]
        base_fee_per_gas = block.get('baseFeePerGas', 0)

        # Get the 'to' address
        to_address = tx_dict.get('to', None)
        if to_address and hasattr(to_address, 'lower'):
            to_address = to_address.lower()

        rows.append([
            tx_hash,
            block_number,
            size,
            compressed_tx_size,
            l1_base_fee,
            l1_blob_base_fee,
            L1_fee,
            gas_price,
            gas_used,
            base_fee_per_gas,
            to_address
        ])

        # Update L1 fee oracle values if needed
        if tx_dict.get('to') == '0x5300000000000000000000000000000000000002':
            input_hex = tx_dict['input'].hex() if hasattr(tx_dict['input'], 'hex') else tx_dict['input']
            if input_hex[:10] == '0x39455d3a':
                l1_base_fee = int.from_bytes(bytes.fromhex(input_hex[2:])[-64:-32], 'big')
                l1_blob_base_fee = int.from_bytes(bytes.fromhex(input_hex[2:])[-32:], 'big')

    tx_df_columns = ['hash', 'block_number', 'size', 'compressed_tx_size', 'l1_base_fee', 'l1_blob_base_fee',
                     'L1_fee', 'gas_price', 'gas_used', 'base_fee_per_gas', 'to']
    tx_df = pd.DataFrame(rows, columns=tx_df_columns)

    # Classify transaction types based on 'to' address
    def classify_tx_type(to_address):
        """Classify transaction type based on destination address"""
        if to_address == ETHERFI_SPEND_ADDRESS:
            return 'etherfi_spend'
        elif to_address == ETHERFI_SWAP_ADDRESS:
            return 'etherfi_swap'
        else:
            return 'other'

    tx_df['tx_type'] = tx_df['to'].apply(classify_tx_type)

    # Add batch_index to each transaction
    print(f"\n  Mapping transactions to batches...")

    def block_to_batch(block_num):
        for idx, row in batch_df.iterrows():
            if row['initial_L2_block_number'] <= block_num < row['initial_L2_block_number'] + row['num_blocks']:
                return idx
        return None

    tx_df['batch_index'] = tx_df['block_number'].apply(block_to_batch)

    print(f"\n  Collected {len(tx_df)} transactions")
    print(f"  Transaction size stats:")
    print(f"    Mean: {tx_df['compressed_tx_size'].mean():.0f} bytes")
    print(f"    Median: {tx_df['compressed_tx_size'].median():.0f} bytes")
    print(f"    P95: {tx_df['compressed_tx_size'].quantile(0.95):.0f} bytes")

    # Report EtherFi transaction counts
    etherfi_spend_count = (tx_df['tx_type'] == 'etherfi_spend').sum()
    etherfi_swap_count = (tx_df['tx_type'] == 'etherfi_swap').sum()
    if etherfi_spend_count > 0 or etherfi_swap_count > 0:
        print(f"\n  EtherFi transaction counts:")
        print(f"    EtherFi Spend: {etherfi_spend_count}")
        print(f"    EtherFi Swap:  {etherfi_swap_count}")

    if start_time:
        print(f"\n✓ Transaction data collection completed in {time.time() - start_time:.1f}s")

    return tx_df


# ============================================================================
# DATA PERSISTENCE
# ============================================================================

def save_data(tx_df, batch_df, start_batch, end_batch):
    """
    Save collected data to pickle file with batch range in filename

    Args:
        tx_df: DataFrame of transactions
        batch_df: DataFrame of batches with L1 cost data
        start_batch: First batch index
        end_batch: Last batch index
    """
    filename = f"galileo_data_batch_{start_batch}_{end_batch}.pkl"
    data = {
        'tx_df': tx_df,
        'batch_df': batch_df,
        'start_batch': start_batch,
        'end_batch': end_batch
    }

    with open(filename, 'wb') as f:
        pickle.dump(data, f)

    print(f"\n✓ Data saved to {filename}")
    print(f"  Batches: {start_batch} - {end_batch} ({end_batch - start_batch + 1} batches)")
    print(f"  Transactions: {len(tx_df)}")


def load_data(start_batch, end_batch):
    """
    Load data from pickle file

    Args:
        start_batch: First batch index
        end_batch: Last batch index

    Returns:
        tx_df, batch_df
    """
    filename = f"galileo_data_batch_{start_batch}_{end_batch}.pkl"

    if not Path(filename).exists():
        raise FileNotFoundError(f"Data file not found: {filename}")

    with open(filename, 'rb') as f:
        data = pickle.load(f)

    print(f"\n✓ Data loaded from {filename}")
    print(f"  Batches: {data['start_batch']} - {data['end_batch']} ({data['end_batch'] - data['start_batch'] + 1} batches)")
    print(f"  Transactions: {len(data['tx_df'])}")

    return data['tx_df'], data['batch_df']


# ============================================================================
# STAGE 1: CALCULATE PENALTY MULTIPLIER
# ============================================================================

def calculate_penalty_multiplier(tx_df, target_penalty=0.1):
    """
    Calculate penalty_multiplier based on transaction size distribution

    Sets penalty_multiplier such that P95 transactions get target_penalty (default 10%)
    from the quadratic term.

    Args:
        tx_df: DataFrame with 'compressed_tx_size' column
        target_penalty: Target penalty ratio at P95 (default 0.1 = 10%)

    Returns:
        penalty_multiplier
    """
    sizes = tx_df['compressed_tx_size'].values

    p50 = np.percentile(sizes, 50)
    p95 = np.percentile(sizes, 95)
    p99 = np.percentile(sizes, 99)

    # For P95 transaction: quadratic_term / linear_term = target_penalty
    # (P95^2 / penalty_multiplier) / P95 = target_penalty
    # penalty_multiplier = P95 / target_penalty
    penalty_multiplier = p95 / target_penalty

    print("=" * 60)
    print("PENALTY MULTIPLIER CALCULATION")
    print("=" * 60)
    print(f"Total transactions: {len(sizes):,}")
    print(f"\nTransaction size distribution:")
    print(f"  P50 (median): {p50:.0f} bytes")
    print(f"  P95:          {p95:.0f} bytes")
    print(f"  P99:          {p99:.0f} bytes")
    print(f"\nTarget penalty at P95: {target_penalty*100:.0f}%")
    print(f"=> penalty_multiplier = {penalty_multiplier:.0f}")
    print("=" * 60)

    return penalty_multiplier


# ============================================================================
# STAGE 2: AGGREGATE BATCH DATA
# ============================================================================

def aggregate_batch_data(tx_df, batch_df, penalty_multiplier):
    """
    Aggregate transaction data by batch with given penalty_multiplier

    Calculates effective_size and equation coefficients for each batch.

    Args:
        tx_df: DataFrame of transactions
        batch_df: DataFrame of batches with L1 costs
        penalty_multiplier: Fixed penalty multiplier

    Returns:
        batch_agg_df: DataFrame with columns:
            - batch_index
            - sum_l1_base_effective: Σ[l1_base_fee × effective_size]
            - sum_blob_base_effective: Σ[l1_blob_base_fee × effective_size]
            - commit_cost: L1 commit cost
            - finalize_cost: L1 finalize cost
            - blob_cost: L1 blob cost
    """
    print("=" * 60)
    print("AGGREGATING BATCH DATA")
    print("=" * 60)

    # Calculate effective_size for all transactions
    tx_df['effective_size'] = (tx_df['compressed_tx_size'] +
                                tx_df['compressed_tx_size']**2 / penalty_multiplier)

    # Calculate coefficients
    tx_df['l1_base_effective'] = tx_df['l1_base_fee'] * tx_df['effective_size']
    tx_df['blob_base_effective'] = tx_df['l1_blob_base_fee'] * tx_df['effective_size']

    # Aggregate by batch
    batch_agg = tx_df.groupby('batch_index').agg({
        'l1_base_effective': 'sum',
        'blob_base_effective': 'sum'
    }).rename(columns={
        'l1_base_effective': 'sum_l1_base_effective',
        'blob_base_effective': 'sum_blob_base_effective'
    })

    # Merge with batch L1 costs
    batch_agg_df = batch_agg.join(batch_df[['commit_cost', 'finalize_cost', 'blob_cost']])

    print(f"Aggregated {len(batch_agg_df)} batches")
    print(f"\nSample batch data:")
    print(batch_agg_df.head())

    return batch_agg_df


# ============================================================================
# STAGE 3: SOLVE FOR SCALARS
# ============================================================================

def solve_scalars(batch_df):
    """
    Solve for commit_scalar and blob_scalar using simple algebraic division

    Sum all batches and divide:
    - commit_scalar = Σ(commit_cost + finalize_cost) / Σ(sum_l1_base_effective)
    - blob_scalar = Σ(blob_cost) / Σ(sum_blob_base_effective)

    Args:
        batch_df: DataFrame with columns:
            - sum_l1_base_effective
            - sum_blob_base_effective
            - commit_cost
            - finalize_cost
            - blob_cost

    Returns:
        commit_scalar, blob_scalar
    """
    total_commit_finalize_cost = (batch_df['commit_cost'] + batch_df['finalize_cost']).sum()
    total_sum_l1_base_effective = batch_df['sum_l1_base_effective'].sum()

    total_blob_cost = batch_df['blob_cost'].sum()
    total_sum_blob_base_effective = batch_df['sum_blob_base_effective'].sum()

    commit_scalar = total_commit_finalize_cost / total_sum_l1_base_effective
    blob_scalar = total_blob_cost / total_sum_blob_base_effective

    print("=" * 60)
    print("SCALAR CALCULATION")
    print("=" * 60)
    print(f"Commit/Finalize:")
    print(f"  Total cost: {total_commit_finalize_cost:,.0f} wei")
    print(f"  Total sum(l1_base × effective_size): {total_sum_l1_base_effective:,.0f}")
    print(f"  => commit_scalar = {commit_scalar:.2f}")
    print()
    print(f"Blob:")
    print(f"  Total cost: {total_blob_cost:,.0f} wei")
    print(f"  Total sum(blob_base × effective_size): {total_sum_blob_base_effective:,.0f}")
    print(f"  => blob_scalar = {blob_scalar:.2f}")
    print("=" * 60)

    return commit_scalar, blob_scalar


# ============================================================================
# VALIDATION & ANALYSIS
# ============================================================================

def analyze_results(commit_scalar, blob_scalar, penalty_multiplier, tx_df, batch_df):
    """
    Analyze the quality of the derived parameters

    Args:
        commit_scalar: Derived commit scalar
        blob_scalar: Derived blob scalar
        penalty_multiplier: Fixed penalty multiplier
        tx_df: Transaction DataFrame
        batch_df: Batch DataFrame

    Returns:
        Analysis results
    """
    print("\n" + "=" * 60)
    print("RESULTS ANALYSIS")
    print("=" * 60)

    # Calculate effective_size
    tx_df['effective_size'] = (tx_df['compressed_tx_size'] +
                                tx_df['compressed_tx_size']**2 / penalty_multiplier)

    # Calculate predicted L1 fees per transaction
    tx_df['predicted_l1_fee'] = ((tx_df['l1_base_fee'] * commit_scalar +
                                   tx_df['l1_blob_base_fee'] * blob_scalar) *
                                  tx_df['effective_size'])

    # Aggregate by batch
    batch_predicted = tx_df.groupby('batch_index')['predicted_l1_fee'].sum()
    batch_actual = batch_df['commit_cost'] + batch_df['finalize_cost'] + batch_df['blob_cost']

    # Calculate errors
    batch_errors = batch_predicted - batch_actual
    batch_relative_errors = batch_errors / batch_actual * 100

    # Statistics
    rmse = np.sqrt(np.mean(batch_errors**2))
    mae = np.mean(np.abs(batch_errors))
    mape = np.mean(np.abs(batch_relative_errors))
    max_error = np.max(np.abs(batch_errors))
    max_relative_error = np.max(np.abs(batch_relative_errors))

    print(f"\nPer-Batch Error Statistics:")
    print(f"  RMSE:                  {rmse:,.0f} wei ({rmse/1e18:.6f} ETH)")
    print(f"  MAE:                   {mae:,.0f} wei ({mae/1e18:.6f} ETH)")
    print(f"  MAPE:                  {mape:.2f}%")
    print(f"  Max absolute error:    {max_error:,.0f} wei ({max_error/1e18:.6f} ETH)")
    print(f"  Max relative error:    {max_relative_error:.2f}%")

    # Total recovery ratio
    total_predicted = batch_predicted.sum()
    total_actual = batch_actual.sum()
    recovery_ratio = total_predicted / total_actual

    print(f"\nTotal Cost Recovery:")
    print(f"  Predicted total: {total_predicted:,.0f} wei ({total_predicted/1e18:.6f} ETH)")
    print(f"  Actual total:    {total_actual:,.0f} wei ({total_actual/1e18:.6f} ETH)")
    print(f"  Recovery ratio:  {recovery_ratio:.4f} ({recovery_ratio*100:.2f}%)")

    return {
        'rmse': rmse,
        'mae': mae,
        'mape': mape,
        'recovery_ratio': recovery_ratio,
        'batch_errors': batch_errors,
        'batch_relative_errors': batch_relative_errors
    }


def analyze_etherfi_penalties(tx_df, penalty_multiplier):
    """
    Analyze penalties for EtherFi transactions

    Calculates the penalty component (quadratic term) for specific transaction types:
    - EtherFi Spend (to: 0x7Ca0b75E67E33c0014325B739A8d019C4FE445F0)
    - EtherFi Swap (to: 0x4dEAa5f2e1CD1A792304d1649EdfA35D565F9346)

    The penalty is the quadratic component: size^2 / penalty_multiplier

    Args:
        tx_df: Transaction DataFrame with 'tx_type' and 'compressed_tx_size' columns
        penalty_multiplier: Penalty multiplier parameter

    Returns:
        Dictionary with penalty statistics by transaction type
    """
    print("\n" + "=" * 60)
    print("ETHERFI PENALTY ANALYSIS")
    print("=" * 60)

    # Filter for EtherFi transactions
    etherfi_spend_txs = tx_df[tx_df['tx_type'] == 'etherfi_spend'].copy()
    etherfi_swap_txs = tx_df[tx_df['tx_type'] == 'etherfi_swap'].copy()

    if len(etherfi_spend_txs) == 0 and len(etherfi_swap_txs) == 0:
        print("\nNo EtherFi transactions found in dataset.")
        print("=" * 60)
        return None

    results = {}

    # Analyze EtherFi Spend transactions
    if len(etherfi_spend_txs) > 0:
        # Calculate penalty (quadratic term) and linear term for each transaction
        etherfi_spend_txs['linear_term'] = etherfi_spend_txs['compressed_tx_size']
        etherfi_spend_txs['penalty_term'] = (etherfi_spend_txs['compressed_tx_size']**2 / penalty_multiplier)
        etherfi_spend_txs['penalty_ratio'] = (etherfi_spend_txs['penalty_term'] /
                                               etherfi_spend_txs['linear_term'])

        print(f"\nEtherFi Spend Transactions (to: {ETHERFI_SPEND_ADDRESS}):")
        print(f"  Count: {len(etherfi_spend_txs)}")
        print(f"\n  Transaction Size:")
        print(f"    Mean:   {etherfi_spend_txs['compressed_tx_size'].mean():.0f} bytes")
        print(f"    Median: {etherfi_spend_txs['compressed_tx_size'].median():.0f} bytes")
        print(f"    Min:    {etherfi_spend_txs['compressed_tx_size'].min():.0f} bytes")
        print(f"    Max:    {etherfi_spend_txs['compressed_tx_size'].max():.0f} bytes")
        print(f"\n  Penalty Term (size² / penalty_multiplier):")
        print(f"    Mean:   {etherfi_spend_txs['penalty_term'].mean():.2f} bytes")
        print(f"    Median: {etherfi_spend_txs['penalty_term'].median():.2f} bytes")
        print(f"    Min:    {etherfi_spend_txs['penalty_term'].min():.2f} bytes")
        print(f"    Max:    {etherfi_spend_txs['penalty_term'].max():.2f} bytes")
        print(f"\n  Penalty Ratio (penalty / linear_term):")
        print(f"    Mean:   {etherfi_spend_txs['penalty_ratio'].mean()*100:.2f}%")
        print(f"    Median: {etherfi_spend_txs['penalty_ratio'].median()*100:.2f}%")
        print(f"    Min:    {etherfi_spend_txs['penalty_ratio'].min()*100:.2f}%")
        print(f"    Max:    {etherfi_spend_txs['penalty_ratio'].max()*100:.2f}%")

        results['etherfi_spend'] = {
            'count': len(etherfi_spend_txs),
            'size_mean': etherfi_spend_txs['compressed_tx_size'].mean(),
            'penalty_mean': etherfi_spend_txs['penalty_term'].mean(),
            'penalty_ratio_mean': etherfi_spend_txs['penalty_ratio'].mean()
        }

    # Analyze EtherFi Swap transactions
    if len(etherfi_swap_txs) > 0:
        # Calculate penalty (quadratic term) and linear term for each transaction
        etherfi_swap_txs['linear_term'] = etherfi_swap_txs['compressed_tx_size']
        etherfi_swap_txs['penalty_term'] = (etherfi_swap_txs['compressed_tx_size']**2 / penalty_multiplier)
        etherfi_swap_txs['penalty_ratio'] = (etherfi_swap_txs['penalty_term'] /
                                              etherfi_swap_txs['linear_term'])

        print(f"\nEtherFi Swap Transactions (to: {ETHERFI_SWAP_ADDRESS}):")
        print(f"  Count: {len(etherfi_swap_txs)}")
        print(f"\n  Transaction Size:")
        print(f"    Mean:   {etherfi_swap_txs['compressed_tx_size'].mean():.0f} bytes")
        print(f"    Median: {etherfi_swap_txs['compressed_tx_size'].median():.0f} bytes")
        print(f"    Min:    {etherfi_swap_txs['compressed_tx_size'].min():.0f} bytes")
        print(f"    Max:    {etherfi_swap_txs['compressed_tx_size'].max():.0f} bytes")
        print(f"\n  Penalty Term (size² / penalty_multiplier):")
        print(f"    Mean:   {etherfi_swap_txs['penalty_term'].mean():.2f} bytes")
        print(f"    Median: {etherfi_swap_txs['penalty_term'].median():.2f} bytes")
        print(f"    Min:    {etherfi_swap_txs['penalty_term'].min():.2f} bytes")
        print(f"    Max:    {etherfi_swap_txs['penalty_term'].max():.2f} bytes")
        print(f"\n  Penalty Ratio (penalty / linear_term):")
        print(f"    Mean:   {etherfi_swap_txs['penalty_ratio'].mean()*100:.2f}%")
        print(f"    Median: {etherfi_swap_txs['penalty_ratio'].median()*100:.2f}%")
        print(f"    Min:    {etherfi_swap_txs['penalty_ratio'].min()*100:.2f}%")
        print(f"    Max:    {etherfi_swap_txs['penalty_ratio'].max()*100:.2f}%")

        results['etherfi_swap'] = {
            'count': len(etherfi_swap_txs),
            'size_mean': etherfi_swap_txs['compressed_tx_size'].mean(),
            'penalty_mean': etherfi_swap_txs['penalty_term'].mean(),
            'penalty_ratio_mean': etherfi_swap_txs['penalty_ratio'].mean()
        }

    print("=" * 60)

    return results


# ============================================================================
# MAIN
# ============================================================================

def main():
    """Main execution logic"""
    import argparse

    start_time = time.time()

    parser = argparse.ArgumentParser(description='Derive Galileo Gas Parameters')
    parser.add_argument('--mode', choices=['collect', 'load'], required=True,
                        help='Mode: collect new data or load from cache')
    parser.add_argument('--n-batches', type=int, default=30,
                        help='Number of batches to collect (default: 30)')
    parser.add_argument('--start-batch', type=int,
                        help='Start batch index (for load mode)')
    parser.add_argument('--end-batch', type=int,
                        help='End batch index (for load mode)')
    parser.add_argument('--target-penalty', type=float, default=0.1,
                        help='Target penalty at P95 (default: 0.1 = 10%%)')
    parser.add_argument('--penalty-multiplier', type=float,
                        help='Fixed penalty multiplier (if not specified, will be calculated from P95)')

    args = parser.parse_args()

    print("\n" + "=" * 60)
    print("GALILEO GAS PARAMETER DERIVATION")
    print("=" * 60)

    # Step 0: Read current on-chain parameters
    current_params = read_current_gas_parameters()

    # Step 1: Collect or load data
    if args.mode == 'collect':
        print(f"\nMode: COLLECT DATA ({args.n_batches} batches)")
        print("=" * 60)

        # Collect batch data
        batch_df = collect_batch_data(n_batches=args.n_batches, start_time=start_time)

        # Collect transaction data
        tx_df = collect_transaction_data(batch_df, start_time=start_time)

        # Save data
        start_batch = batch_df.index[0]
        end_batch = batch_df.index[-1]
        save_data(tx_df, batch_df, start_batch, end_batch)

    else:  # load mode
        print(f"\nMode: LOAD DATA (batches {args.start_batch}-{args.end_batch})")
        print("=" * 60)

        if args.start_batch is None or args.end_batch is None:
            raise ValueError("--start-batch and --end-batch required for load mode")

        tx_df, batch_df = load_data(args.start_batch, args.end_batch)

    # Step 2: Use fixed penalty_multiplier if provided, otherwise calculate it
    if args.penalty_multiplier is not None:
        penalty_multiplier = args.penalty_multiplier
        print("=" * 60)
        print("USING FIXED PENALTY MULTIPLIER")
        print("=" * 60)
        print(f"penalty_multiplier = {penalty_multiplier:.0f}")
        print("=" * 60)
    else:
        penalty_multiplier = calculate_penalty_multiplier(tx_df, target_penalty=args.target_penalty)

    # Step 3: Aggregate batch data
    batch_agg_df = aggregate_batch_data(tx_df, batch_df, penalty_multiplier)

    # Step 4: Solve for scalars
    commit_scalar, blob_scalar = solve_scalars(batch_agg_df)

    # Step 5: Analyze results
    results = analyze_results(commit_scalar, blob_scalar, penalty_multiplier, tx_df, batch_df)

    # Step 6: Analyze EtherFi penalties
    etherfi_results = analyze_etherfi_penalties(tx_df, penalty_multiplier)

    # Final summary
    print("\n" + "=" * 60)
    print("FINAL RESULTS")
    print("=" * 60)
    print(f"\nDerived Parameters:")
    print(f"  commit_scalar:        {commit_scalar:.2f}")
    print(f"  blob_scalar:          {blob_scalar:.2f}")
    print(f"  penalty_multiplier:   {penalty_multiplier:.0f}")
    print(f"\nFit Quality:")
    print(f"  MAPE:                 {results['mape']:.2f}%")
    print(f"  Recovery ratio:       {results['recovery_ratio']*100:.2f}%")
    print("=" * 60)

    return {
        'commit_scalar': commit_scalar,
        'blob_scalar': blob_scalar,
        'penalty_multiplier': penalty_multiplier,
        'tx_df': tx_df,
        'batch_df': batch_df,
        'batch_agg_df': batch_agg_df,
        'results': results,
        'etherfi_results': etherfi_results
    }


if __name__ == "__main__":
    main()
