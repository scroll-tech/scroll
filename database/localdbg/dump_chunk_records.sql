-- Create a file with INSERT statements for the specific records
\o chunk_export.sql
\t on
\a on
SELECT 'INSERT INTO chunk ("index", hash, start_block_number, end_block_number,
       start_block_hash, end_block_hash, total_l1_messages_popped_before,
       total_l1_messages_popped_in_chunk, start_block_time,
       parent_chunk_hash, state_root, parent_chunk_state_root,
       withdraw_root, total_l2_tx_gas, total_l2_tx_num,
       total_l1_commit_calldata_size, total_l1_commit_gas,
       enable_compress, prev_l1_message_queue_hash, post_l1_message_queue_hash
       ) VALUES (' || 
       quote_literal("index") || ', ' || 
       quote_literal(hash) || ', ' || 
       quote_literal(start_block_number) || ', ' || 
       quote_literal(end_block_number) || ', ' ||
       quote_literal(start_block_hash) || ', ' ||
       quote_literal(end_block_hash) || ', ' ||
       quote_literal(total_l1_messages_popped_before) || ', ' ||
       quote_literal(total_l1_messages_popped_in_chunk) || ', ' ||
       quote_literal(start_block_time) || ', ' ||
       quote_literal(parent_chunk_hash) || ', ' ||
       quote_literal(state_root) || ', ' ||
       quote_literal(parent_chunk_state_root) || ', ' ||
       quote_literal(withdraw_root) || ', ' ||
       quote_literal(total_l2_tx_gas) || ', ' ||
       quote_literal(total_l2_tx_num) || ', ' ||
       quote_literal(total_l1_commit_calldata_size) || ', ' ||
       quote_literal(total_l1_commit_gas) || ', ' ||
       quote_literal(enable_compress) || ', ' ||
       quote_literal(prev_l1_message_queue_hash) || ', ' ||
       quote_literal(post_l1_message_queue_hash) ||
       ');'
FROM chunk 
ORDER BY "index" DESC 
LIMIT 10;
\t off
\a off
\o