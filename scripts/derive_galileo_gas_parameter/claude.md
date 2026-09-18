# Project Description

This is a Scroll L2 blockchain gas fee parameter calculation project for deriving Scroll gas fee parameters.

# Environment Setup

Claude is executed within a pipenv shell environment, so all packages should already be installed.

# Guidelines

- All Python script comments must be in English
- When executing `derive_galileo_gas_parameter.py`, output should be redirected to a fixed log file
  - Use the `-u` option to disable output buffering for real-time logging
  - Always use `/tmp/derive_galileo_gas_parameter.log` as the log file
  - Example: `python -u derive_galileo_gas_parameter.py --mode collect --n-batches 5 > /tmp/derive_galileo_gas_parameter.log 2>&1 &`

# Python Script Execution

## Command Format

```bash
python -u derive_galileo_gas_parameter.py [OPTIONS] > /tmp/derive_galileo_gas_parameter.log 2>&1 &
```

## Options

- `--mode {collect,load}` (required): Operation mode
  - `collect`: Collect new data from blockchain
  - `load`: Load data from cached pickle file

- `--n-batches N`: Number of batches to collect (default: 30)
  - Only used in `collect` mode

- `--start-batch N`: Start batch index (required for `load` mode)

- `--end-batch N`: End batch index (required for `load` mode)

- `--target-penalty FLOAT`: Target penalty at P95 (default: 0.1 = 10%)

- `--penalty-multiplier FLOAT`: Fixed penalty multiplier (optional, will be calculated from P95 if not specified)

## Examples

Collect data for 3 batches:
```bash
python -u derive_galileo_gas_parameter.py --mode collect --n-batches 3 > /tmp/derive_galileo_gas_parameter.log 2>&1 &
```

Load cached data:
```bash
python -u derive_galileo_gas_parameter.py --mode load --start-batch 12345 --end-batch 12347 > /tmp/derive_galileo_gas_parameter.log 2>&1 &
```

Collect with custom target penalty:
```bash
python -u derive_galileo_gas_parameter.py --mode collect --n-batches 30 --target-penalty 0.15 > /tmp/derive_galileo_gas_parameter.log 2>&1 &
```

## Monitoring Execution

Check log output in real-time:
```bash
tail -f /tmp/derive_galileo_gas_parameter.log
```

View complete log:
```bash
cat /tmp/derive_galileo_gas_parameter.log
```

