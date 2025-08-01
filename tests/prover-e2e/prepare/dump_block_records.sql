-- Create a file with INSERT statements for the specific records
-- We noticed that transactions is not used within coordinator so simply
-- replace it with empty record
\o block_export.sql
\t on
\a
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
WHERE number >= 16523677 and number <= 16523700
ORDER BY number ASC;
\t off
\a
\o