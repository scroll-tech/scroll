-- Create a file with INSERT statements for the specific records
\o 00100_import_blocks.sql
\t on
\a
-- Write header comment
SELECT '-- +goose Up';
SELECT '-- +goose StatementBegin';
SELECT '';

SELECT 'INSERT INTO l2_block (number, hash, parent_hash, header, withdraw_root,
       state_root, tx_num, gas_used, block_timestamp, row_consumption,
       chunk_hash, transactions
       ) VALUES (' || 
       quote_literal(number) || ', ' || 
       quote_literal(hash) || ', ' || 
       quote_literal(parent_hash) || ', ' || 
       quote_literal(header) || ', ' || 
       quote_literal(withdraw_root) || ', ' ||
       quote_literal(state_root) || ', ' ||
       quote_literal(tx_num) || ', ' ||
       quote_literal(gas_used) || ', ' ||
       quote_literal(block_timestamp) || ', ' ||
       quote_literal(row_consumption) || ', ' ||
       quote_literal(chunk_hash) || ', ' ||
       quote_literal(transactions) ||
       ');'
FROM l2_block 
WHERE number >= 1 and number <= 49
ORDER BY number ASC;

-- Write footer
SELECT '';
SELECT '-- +goose StatementEnd';
SELECT '-- +goose Down';
SELECT '-- +goose StatementBegin';
SELECT 'DELETE FROM l2_block;';
SELECT '-- +goose StatementEnd';

\t off
\a
\o