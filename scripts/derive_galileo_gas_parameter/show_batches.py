#!/usr/bin/env python3
"""
Show batch IDs in a saved data file
"""

import pickle
import sys
from pathlib import Path

def show_batches(filename):
    """Show batch IDs in the data file"""

    if not Path(filename).exists():
        print(f"Error: File not found: {filename}")
        return

    try:
        with open(filename, 'rb') as f:
            data = pickle.load(f)

        batch_df = data['batch_df']
        tx_df = data['tx_df']

        print(f"\n{'='*60}")
        print(f"File: {filename}")
        print(f"{'='*60}")
        print(f"\nBatch range: {data['start_batch']} - {data['end_batch']}")
        print(f"Total batches: {len(batch_df)}")
        print(f"Total transactions: {len(tx_df)}")

        print(f"\nBatch IDs:")
        batch_ids = sorted(batch_df.index.tolist())

        # Print batch IDs in a compact format
        for i in range(0, len(batch_ids), 10):
            batch_subset = batch_ids[i:i+10]
            print(f"  {', '.join(map(str, batch_subset))}")

        # Show detailed information for each batch
        print(f"\n{'='*60}")
        print("BATCH DETAILS")
        print(f"{'='*60}")

        for batch_id in sorted(batch_df.index.tolist()):
            batch_row = batch_df.loc[batch_id]
            print(f"\n{'─'*60}")
            print(f"Batch {int(batch_id)}")
            print(f"{'─'*60}")
            print(f"  versioned_hash: {batch_row['versioned_hash']}")
            print(f"  commit_cost: {batch_row['commit_cost'] / 1e9:.9f} gwei")
            print(f"  blob_cost: {batch_row['blob_cost'] / 1e9:.9f} gwei")
            print(f"  finalize_cost: {batch_row['finalize_cost'] / 1e9:.9f} gwei")

            # Get transactions for this batch
            batch_txs = tx_df[tx_df['batch_index'] == batch_id]
            print(f"\n  Transactions ({len(batch_txs)} total):")

            if len(batch_txs) > 0:
                for idx, tx in batch_txs.iterrows():
                    print(f"\n    Transaction {idx}:")
                    print(f"      hash: {tx['hash']}")
                    print(f"      block_number: {tx['block_number']}")
                    print(f"      size: {tx['size']}")
                    print(f"      compressed_tx_size: {tx['compressed_tx_size']}")
                    print(f"      l1_base_fee: {tx['l1_base_fee'] / 1e9:.9f} gwei")
                    print(f"      l1_blob_base_fee: {tx['l1_blob_base_fee'] / 1e9:.9f} gwei")
            else:
                print("      (No transactions)")

        print(f"\n{'='*60}\n")

    except Exception as e:
        print(f"Error reading file: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python show_batches.py <data_file.pkl>")
        print("\nExample: python show_batches.py galileo_data_batch_491125_491136.pkl")
        sys.exit(1)

    show_batches(sys.argv[1])
