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
import subprocess
import json
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

# Validate that environment variables are set
if not mainnet_url:
    raise ValueError("MAINNET_URL environment variable is not set. Please set it to your Ethereum mainnet RPC endpoint.")
if not scroll_url:
    raise ValueError("SCROLL_URL environment variable is not set. Please set it to your Scroll RPC endpoint.")

# Scroll Rollup Explorer API
ROLLUP_EXPLORER_API = "https://mainnet-api-re.scroll.io/api"

# EtherFi contract addresses (lowercase for comparison)
ETHERFI_SPEND_ADDRESS = "0x7ca0b75e67e33c0014325b739a8d019c4fe445f0"
ETHERFI_SWAP_ADDRESS = "0x4deaa5f2e1cd1a792304d1649edfa35d565f9346"

# ============================================================================
# ABIs
# ============================================================================

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

def curl_get_json(url, max_retries=3):
    """Fetch JSON from URL using curl (bypasses Python SSL issues with VPN)"""
    for attempt in range(max_retries):
        result = subprocess.run(
            ['curl', '-s', '--fail', url],
            capture_output=True, text=True, timeout=30
        )
        if result.returncode == 0 and result.stdout.strip():
            return json.loads(result.stdout)
        if attempt < max_retries - 1:
            time.sleep(1)
    raise RuntimeError(f"curl failed for {url} after {max_retries} attempts: {result.stderr}")


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
# DATA COLLECTION FUNCTIONS
# ============================================================================

def collect_batch_data(n_batches=30, start_time=None):
    """
    Collect batch data using Scroll Rollup Explorer API and L1 transaction receipts.

    Args:
        n_batches: Number of batches to collect
        start_time: Script start time for elapsed time calculation

    Returns:
        batch_df: DataFrame with columns:
            - index (batch_index)
            - initial_L2_block_number
            - num_blocks
            - commit_cost
            - finalize_cost
            - blob_cost
    """
    print("=" * 60)
    print(f"COLLECTING BATCH DATA ({n_batches} batches)")
    print("=" * 60)

    # Step 1: Get latest finalized batch index from Rollup Explorer API
    print(f"\nStep 1: Getting latest finalized batch index...")
    data = curl_get_json(f"{ROLLUP_EXPLORER_API}/last_batch_indexes")
    latest_finalized = data['finalized_index']
    print(f"  Latest finalized batch: {latest_finalized}")

    batch_indices = list(range(latest_finalized - n_batches + 1, latest_finalized + 1))
    print(f"  Collecting batches: {batch_indices[0]} - {batch_indices[-1]}")

    # Step 2: Fetch batch info from Rollup Explorer API
    print(f"\nStep 2: Fetching batch info from Rollup Explorer API...")

    batch_infos = {}
    for i, batch_index in enumerate(batch_indices):
        data = curl_get_json(f"{ROLLUP_EXPLORER_API}/batch?index={batch_index}")
        batch_infos[data['batch']['index']] = data['batch']
        if (i + 1) % 50 == 0 or (i + 1) == n_batches:
            print(f"  Fetched {i + 1}/{n_batches} batch infos")

    # Step 3: Fetch L1 transaction receipts for cost calculation
    print(f"\nStep 3: Fetching L1 transaction data...")

    # Collect unique tx hashes
    commit_tx_hashes = set()
    finalize_tx_hashes = set()
    for info in batch_infos.values():
        commit_tx_hashes.add(info['commit_tx_hash'])
        finalize_tx_hashes.add(info['finalize_tx_hash'])

    print(f"  Unique commit txs: {len(commit_tx_hashes)}, finalize txs: {len(finalize_tx_hashes)}")

    # Count batches per finalize tx for amortization
    finalize_tx_batch_count = {}
    for info in batch_infos.values():
        h = info['finalize_tx_hash']
        finalize_tx_batch_count[h] = finalize_tx_batch_count.get(h, 0) + 1

    # For commit txs: fetch tx (for blobVersionedHashes count) + receipt
    # For finalize txs: fetch receipt only
    commit_tx_cache = {}   # tx_hash -> (tx, receipt)
    finalize_rx_cache = {} # tx_hash -> receipt

    def fetch_commit_tx_data(tx_hash):
        tx = w3.eth.get_transaction(tx_hash)
        receipt = w3.eth.get_transaction_receipt(tx_hash)
        return ('commit', tx_hash, tx, receipt)

    def fetch_finalize_receipt(tx_hash):
        receipt = w3.eth.get_transaction_receipt(tx_hash)
        return ('finalize', tx_hash, None, receipt)

    all_fetches = []
    for h in commit_tx_hashes:
        all_fetches.append(('commit', h))
    for h in finalize_tx_hashes:
        all_fetches.append(('finalize', h))

    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = []
        for type_, h in all_fetches:
            if type_ == 'commit':
                futures.append(executor.submit(fetch_commit_tx_data, h))
            else:
                futures.append(executor.submit(fetch_finalize_receipt, h))

        completed = 0
        total = len(futures)
        for future in as_completed(futures):
            type_, tx_hash, tx, receipt = future.result()
            if type_ == 'commit':
                commit_tx_cache[tx_hash] = (tx, receipt)
            else:
                finalize_rx_cache[tx_hash] = receipt
            completed += 1
            if completed % 50 == 0 or completed == total:
                print(f"  Fetched {completed}/{total} L1 transactions")

    # Step 4: Build batch data
    print(f"\nStep 4: Building batch data...")
    batch_list = []
    for batch_index in sorted(batch_infos.keys()):
        info = batch_infos[batch_index]

        commit_tx, commit_receipt = commit_tx_cache[info['commit_tx_hash']]
        finalize_receipt = finalize_rx_cache[info['finalize_tx_hash']]

        # Amortize commit cost by number of blobs (= number of batches in commit tx)
        n_blobs = len(commit_tx.blobVersionedHashes)
        commit_cost = (commit_receipt['gasUsed'] * commit_receipt['effectiveGasPrice']) / n_blobs
        blob_cost = (commit_receipt['blobGasPrice'] * commit_receipt['blobGasUsed']) / n_blobs

        # Amortize finalize cost by number of batches sharing the same finalize tx
        n_finalize = finalize_tx_batch_count[info['finalize_tx_hash']]
        finalize_cost = (finalize_receipt['gasUsed'] * finalize_receipt['effectiveGasPrice']) / n_finalize

        initial_L2_block_number = info['start_block_number']
        num_blocks = info['end_block_number'] - info['start_block_number'] + 1

        batch_list.append([
            batch_index, initial_L2_block_number, num_blocks,
            commit_cost, finalize_cost, blob_cost
        ])

    # Create DataFrame
    column_names = ['index', 'initial_L2_block_number', 'num_blocks',
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
    max_workers = 100  # Higher limit for paid RPC endpoint
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

    # Build rows with oracle fee tracking
    print(f"\n  Building transaction dataframe...")
    rows = []
    for block_number, tx_dict in all_txs:
        tx_hash = tx_dict['hash'].hex() if hasattr(tx_dict['hash'], 'hex') else tx_dict['hash']
        size, compressed_tx_size = tx_sizes[tx_hash]

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
            to_address
        ])

        # Update L1 fee oracle values if needed
        if tx_dict.get('to') == '0x5300000000000000000000000000000000000002':
            input_hex = tx_dict['input'].hex() if hasattr(tx_dict['input'], 'hex') else tx_dict['input']
            if input_hex[:10] == '0x39455d3a':
                l1_base_fee = int.from_bytes(bytes.fromhex(input_hex[2:])[-64:-32], 'big')
                l1_blob_base_fee = int.from_bytes(bytes.fromhex(input_hex[2:])[-32:], 'big')

    tx_df_columns = ['hash', 'block_number', 'size', 'compressed_tx_size', 'l1_base_fee', 'l1_blob_base_fee', 'to']
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

    # Build block_number -> batch_index lookup for O(1) mapping
    block_to_batch = {}
    for idx, row in batch_df.iterrows():
        start = int(row['initial_L2_block_number'])
        count = int(row['num_blocks'])
        for b in range(start, start + count):
            block_to_batch[b] = idx

    tx_df['batch_index'] = tx_df['block_number'].map(block_to_batch)

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
# PARAMETER COMPARISON ANALYSIS
# ============================================================================

def compare_parameters(current_params, commit_scalar, blob_scalar, penalty_multiplier, tx_df, batch_df):
    """
    Compare derived parameters against current on-chain parameters.

    Computes per-tx L1 fees under both old and new parameter sets using:
        effective_size = compressed_tx_size + compressed_tx_size² / penalty_factor
        L1_fee = (commit_scalar * l1_base_fee + blob_scalar * l1_blob_base_fee) * effective_size

    Args:
        current_params: Dict with current on-chain parameters from read_current_gas_parameters()
        commit_scalar: Newly derived commit scalar
        blob_scalar: Newly derived blob scalar
        penalty_multiplier: Newly derived penalty multiplier
        tx_df: Transaction DataFrame
        batch_df: Batch DataFrame with L1 costs
    """
    print("\n" + "=" * 60)
    print("PARAMETER COMPARISON ANALYSIS")
    print("=" * 60)

    # L1 Gas Price Distribution
    print("\n" + "-" * 60)
    print("L1 Gas Price Distribution (as seen by L2 fee oracle)")
    print("-" * 60)

    l1_base = tx_df['l1_base_fee']
    l1_blob = tx_df['l1_blob_base_fee']

    print(f"\n  l1_base_fee (wei):")
    print(f"    Min:    {l1_base.min():>15,}  ({l1_base.min()/1e9:.4f} gwei)")
    print(f"    P5:     {l1_base.quantile(0.05):>15,.0f}  ({l1_base.quantile(0.05)/1e9:.4f} gwei)")
    print(f"    Median: {l1_base.median():>15,.0f}  ({l1_base.median()/1e9:.4f} gwei)")
    print(f"    Mean:   {l1_base.mean():>15,.0f}  ({l1_base.mean()/1e9:.4f} gwei)")
    print(f"    P95:    {l1_base.quantile(0.95):>15,.0f}  ({l1_base.quantile(0.95)/1e9:.4f} gwei)")
    print(f"    Max:    {l1_base.max():>15,}  ({l1_base.max()/1e9:.4f} gwei)")
    print(f"    Max/Min ratio: {l1_base.max()/l1_base.min():.1f}x")

    print(f"\n  l1_blob_base_fee (wei):")
    print(f"    Min:    {l1_blob.min():>15,}  ({l1_blob.min()/1e9:.4f} gwei)")
    print(f"    P5:     {l1_blob.quantile(0.05):>15,.0f}  ({l1_blob.quantile(0.05)/1e9:.4f} gwei)")
    print(f"    Median: {l1_blob.median():>15,.0f}  ({l1_blob.median()/1e9:.4f} gwei)")
    print(f"    Mean:   {l1_blob.mean():>15,.0f}  ({l1_blob.mean()/1e9:.4f} gwei)")
    print(f"    P95:    {l1_blob.quantile(0.95):>15,.0f}  ({l1_blob.quantile(0.95)/1e9:.4f} gwei)")
    print(f"    Max:    {l1_blob.max():>15,}  ({l1_blob.max()/1e9:.4f} gwei)")
    print(f"    Max/Min ratio: {l1_blob.max()/l1_blob.min():.1f}x")

    # Extract current (old) parameters
    old_commit_scalar = current_params['commit_scalar']
    old_blob_scalar = current_params['blob_scalar']
    old_penalty_factor = current_params['penalty_factor']

    new_commit_scalar = commit_scalar
    new_blob_scalar = blob_scalar
    new_penalty_factor = penalty_multiplier

    # --- Analysis 1: Fee Comparison (Old vs New) ---
    print("\n" + "-" * 60)
    print("Analysis 1: Fee Comparison (Old vs New)")
    print("-" * 60)

    print(f"\n  Parameter Comparison:")
    print(f"  {'Parameter':<25} {'Current':>15} {'New':>15} {'Change':>15}")
    print(f"  {'-'*25} {'-'*15} {'-'*15} {'-'*15}")
    print(f"  {'commit_scalar':<25} {old_commit_scalar:>15.4f} {new_commit_scalar:>15.4f} {new_commit_scalar - old_commit_scalar:>+15.4f}")
    print(f"  {'blob_scalar':<25} {old_blob_scalar:>15.4f} {new_blob_scalar:>15.4f} {new_blob_scalar - old_blob_scalar:>+15.4f}")
    print(f"  {'penalty_factor':<25} {old_penalty_factor:>15.0f} {new_penalty_factor:>15.0f} {new_penalty_factor - old_penalty_factor:>+15.0f}")

    # Calculate effective_size and fees under both parameter sets
    df = tx_df.copy()

    df['old_effective_size'] = df['compressed_tx_size'] + df['compressed_tx_size']**2 / old_penalty_factor
    df['new_effective_size'] = df['compressed_tx_size'] + df['compressed_tx_size']**2 / new_penalty_factor

    df['old_fee'] = (old_commit_scalar * df['l1_base_fee'] + old_blob_scalar * df['l1_blob_base_fee']) * df['old_effective_size']
    df['new_fee'] = (new_commit_scalar * df['l1_base_fee'] + new_blob_scalar * df['l1_blob_base_fee']) * df['new_effective_size']

    df['fee_change'] = df['new_fee'] - df['old_fee']
    df['fee_change_pct'] = df['fee_change'] / df['old_fee'] * 100

    print(f"\n  Per-Transaction Fee Change Statistics:")
    print(f"    {'Metric':<20} {'Absolute (wei)':>20} {'Percentage':>15}")
    print(f"    {'-'*20} {'-'*20} {'-'*15}")
    print(f"    {'Mean':<20} {df['fee_change'].mean():>+20,.0f} {df['fee_change_pct'].mean():>+14.2f}%")
    print(f"    {'Median':<20} {df['fee_change'].median():>+20,.0f} {df['fee_change_pct'].median():>+14.2f}%")
    print(f"    {'P95':<20} {df['fee_change'].quantile(0.95):>+20,.0f} {df['fee_change_pct'].quantile(0.95):>+14.2f}%")
    print(f"    {'P99':<20} {df['fee_change'].quantile(0.99):>+20,.0f} {df['fee_change_pct'].quantile(0.99):>+14.2f}%")

    paying_more = (df['fee_change'] > 0).sum()
    paying_less = (df['fee_change'] < 0).sum()
    paying_same = (df['fee_change'] == 0).sum()
    total_txs = len(df)

    print(f"\n  Fee Direction:")
    print(f"    Paying more: {paying_more:>8,} ({paying_more/total_txs*100:>6.2f}%)")
    print(f"    Paying less: {paying_less:>8,} ({paying_less/total_txs*100:>6.2f}%)")
    print(f"    No change:   {paying_same:>8,} ({paying_same/total_txs*100:>6.2f}%)")

    total_old_fee = df['old_fee'].sum()
    total_new_fee = df['new_fee'].sum()
    total_change = total_new_fee - total_old_fee

    print(f"\n  Aggregate Total Fee:")
    print(f"    Old total: {total_old_fee:>25,.0f} wei ({total_old_fee/1e18:.6f} ETH)")
    print(f"    New total: {total_new_fee:>25,.0f} wei ({total_new_fee/1e18:.6f} ETH)")
    print(f"    Change:    {total_change:>+25,.0f} wei ({total_change/total_old_fee*100:>+.2f}%)")

    # --- Analysis 2: Fee Change by Transaction Size Group ---
    print("\n" + "-" * 60)
    print("Analysis 2: Fee Change by Transaction Size Group")
    print("-" * 60)

    p50_size = df['compressed_tx_size'].quantile(0.50)
    p95_size = df['compressed_tx_size'].quantile(0.95)

    def size_group(size):
        if size < p50_size:
            return 'Small (< P50)'
        elif size <= p95_size:
            return 'Medium (P50-P95)'
        else:
            return 'Large (> P95)'

    df['size_group'] = df['compressed_tx_size'].apply(size_group)

    # Calculate penalty terms for both parameter sets
    df['old_penalty_term'] = df['compressed_tx_size']**2 / old_penalty_factor
    df['new_penalty_term'] = df['compressed_tx_size']**2 / new_penalty_factor

    group_order = ['Small (< P50)', 'Medium (P50-P95)', 'Large (> P95)']
    print(f"\n  Size thresholds: P50 = {p50_size:.0f} bytes, P95 = {p95_size:.0f} bytes\n")

    for group_name in group_order:
        g = df[df['size_group'] == group_name]
        if len(g) == 0:
            continue

        print(f"  {group_name}:")
        print(f"    Count:          {len(g):,}")
        print(f"    Size range:     {g['compressed_tx_size'].min():.0f} - {g['compressed_tx_size'].max():.0f} bytes")
        print(f"    Old fee:        mean={g['old_fee'].mean():,.0f}  median={g['old_fee'].median():,.0f} wei")
        print(f"    New fee:        mean={g['new_fee'].mean():,.0f}  median={g['new_fee'].median():,.0f} wei")
        print(f"    Fee change:     mean={g['fee_change'].mean():+,.0f}  median={g['fee_change'].median():+,.0f} wei")
        print(f"    Fee change %%:   mean={g['fee_change_pct'].mean():+.2f}%%  median={g['fee_change_pct'].median():+.2f}%%")
        print(f"    Old penalty/eff: mean={((g['old_penalty_term'] / g['old_effective_size']) * 100).mean():.2f}%%")
        print(f"    New penalty/eff: mean={((g['new_penalty_term'] / g['new_effective_size']) * 100).mean():.2f}%%")
        print()

    # --- Analysis 3: Cost Recovery by Batch ---
    print("-" * 60)
    print("Analysis 3: Cost Recovery by Batch")
    print("-" * 60)

    batch_old_fees = df.groupby('batch_index')['old_fee'].sum()
    batch_new_fees = df.groupby('batch_index')['new_fee'].sum()

    batch_actual_cost = batch_df['commit_cost'] + batch_df['finalize_cost'] + batch_df['blob_cost']

    # Align indices
    common_idx = batch_old_fees.index.intersection(batch_actual_cost.index)
    batch_old_fees = batch_old_fees.loc[common_idx]
    batch_new_fees = batch_new_fees.loc[common_idx]
    batch_actual_cost = batch_actual_cost.loc[common_idx]

    old_recovery_rate = batch_old_fees / batch_actual_cost
    new_recovery_rate = batch_new_fees / batch_actual_cost

    print(f"\n  Recovery Rate Statistics:")
    print(f"    {'Metric':<15} {'Old Params':>15} {'New Params':>15}")
    print(f"    {'-'*15} {'-'*15} {'-'*15}")
    print(f"    {'Mean':<15} {old_recovery_rate.mean():>14.4f}x {new_recovery_rate.mean():>14.4f}x")
    print(f"    {'Median':<15} {old_recovery_rate.median():>14.4f}x {new_recovery_rate.median():>14.4f}x")
    print(f"    {'Min':<15} {old_recovery_rate.min():>14.4f}x {new_recovery_rate.min():>14.4f}x")
    print(f"    {'Max':<15} {old_recovery_rate.max():>14.4f}x {new_recovery_rate.max():>14.4f}x")

    old_over = (old_recovery_rate > 1).sum()
    old_under = (old_recovery_rate < 1).sum()
    new_over = (new_recovery_rate > 1).sum()
    new_under = (new_recovery_rate < 1).sum()

    print(f"\n  Over/Under Recovery:")
    print(f"    Old params: {old_over} over, {old_under} under (of {len(common_idx)} batches)")
    print(f"    New params: {new_over} over, {new_under} under (of {len(common_idx)} batches)")

    # Per-batch table (show all if <= 30 batches)
    if len(common_idx) <= 30:
        print(f"\n  Per-Batch Detail:")
        print(f"    {'Batch':>8} {'Actual Cost':>18} {'Old Fees':>18} {'Old Rate':>10} {'New Fees':>18} {'New Rate':>10}")
        print(f"    {'-'*8} {'-'*18} {'-'*18} {'-'*10} {'-'*18} {'-'*10}")
        for idx in sorted(common_idx):
            print(f"    {idx:>8} {batch_actual_cost.loc[idx]:>18,.0f} {batch_old_fees.loc[idx]:>18,.0f} {old_recovery_rate.loc[idx]:>9.4f}x {batch_new_fees.loc[idx]:>18,.0f} {new_recovery_rate.loc[idx]:>9.4f}x")

    # Aggregate totals
    total_actual = batch_actual_cost.sum()
    total_old = batch_old_fees.sum()
    total_new = batch_new_fees.sum()

    print(f"\n  Aggregate Totals:")
    print(f"    Actual L1 cost: {total_actual:>25,.0f} wei ({total_actual/1e18:.6f} ETH)")
    print(f"    Old fee total:  {total_old:>25,.0f} wei ({total_old/1e18:.6f} ETH) ({total_old/total_actual:.4f}x)")
    print(f"    New fee total:  {total_new:>25,.0f} wei ({total_new/1e18:.6f} ETH) ({total_new/total_actual:.4f}x)")

    # --- Analysis 4: Penalty Proportion ---
    print("\n" + "-" * 60)
    print("Analysis 4: Penalty Proportion")
    print("-" * 60)

    df['old_penalty_pct'] = df['old_penalty_term'] / df['old_effective_size'] * 100
    df['new_penalty_pct'] = df['new_penalty_term'] / df['new_effective_size'] * 100

    print(f"\n  Overall Penalty Proportion (penalty_term / effective_size × 100):")
    print(f"    {'Metric':<15} {'Old Params':>15} {'New Params':>15}")
    print(f"    {'-'*15} {'-'*15} {'-'*15}")
    print(f"    {'Mean':<15} {df['old_penalty_pct'].mean():>14.2f}% {df['new_penalty_pct'].mean():>14.2f}%")
    print(f"    {'Median':<15} {df['old_penalty_pct'].median():>14.2f}% {df['new_penalty_pct'].median():>14.2f}%")
    print(f"    {'P95':<15} {df['old_penalty_pct'].quantile(0.95):>14.2f}% {df['new_penalty_pct'].quantile(0.95):>14.2f}%")
    print(f"    {'P99':<15} {df['old_penalty_pct'].quantile(0.99):>14.2f}% {df['new_penalty_pct'].quantile(0.99):>14.2f}%")

    print(f"\n  Per Size Group:")
    for group_name in group_order:
        g = df[df['size_group'] == group_name]
        if len(g) == 0:
            continue

        print(f"\n    {group_name} (n={len(g):,}):")
        print(f"      {'Metric':<15} {'Old Params':>15} {'New Params':>15}")
        print(f"      {'-'*15} {'-'*15} {'-'*15}")
        print(f"      {'Mean':<15} {g['old_penalty_pct'].mean():>14.2f}% {g['new_penalty_pct'].mean():>14.2f}%")
        print(f"      {'Median':<15} {g['old_penalty_pct'].median():>14.2f}% {g['new_penalty_pct'].median():>14.2f}%")
        print(f"      {'P95':<15} {g['old_penalty_pct'].quantile(0.95):>14.2f}% {g['new_penalty_pct'].quantile(0.95):>14.2f}%")
        print(f"      {'P99':<15} {g['old_penalty_pct'].quantile(0.99):>14.2f}% {g['new_penalty_pct'].quantile(0.99):>14.2f}%")

    print("\n" + "=" * 60)


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

    # Step 7: Compare with current on-chain parameters
    compare_parameters(current_params, commit_scalar, blob_scalar, penalty_multiplier, tx_df, batch_df)

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
