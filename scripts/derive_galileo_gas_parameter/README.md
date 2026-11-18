# Galileo Gas Parameter Derivation

Derives L1 fee parameters (`commit_scalar`, `blob_scalar`, `penalty_multiplier`) for Scroll's Galileo upgrade by analyzing historical on-chain data.

## Prerequisites

- Python 3.10+
- pipenv
- gcc and python3-devel

```bash
# Ubuntu/Debian
sudo apt install python3-dev gcc
# Fedora/RHEL
sudo dnf install python3-devel gcc
```

## Installation

```bash
cd scripts/derive_galileo_gas_parameter
pipenv install
```

## Running

### 1. Create `.env` file

```bash
SCROLL_URL=https://rpc.scroll.io
MAINNET_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY
BEACON_URL=https://eth-mainnet-beacon.g.alchemy.com/v2/YOUR_API_KEY
```

### 2. Usage

```bash
# Collect data from 30 recent batches
pipenv run python -u derive_galileo_gas_parameter.py --mode collect --n-batches 30

# Load previously cached data
pipenv run python -u derive_galileo_gas_parameter.py --mode load --start-batch 494041 --end-batch 494070
```

**Options:**
- `--mode {collect,load}` - Collect new data or load from cache (required)
- `--n-batches N` - Number of batches to collect (default: 30)
- `--start-batch N` / `--end-batch N` - Batch range for load mode
- `--target-penalty FLOAT` - Target penalty at P95 (default: 0.1 = 10%)
- `--penalty-multiplier FLOAT` - Use fixed penalty multiplier instead of calculating

### 3. Cached Data

Collected data is saved to `galileo_data_batch_{start}_{end}.pkl` for re-analysis without re-fetching from RPC. Use `--mode load` to reload.
