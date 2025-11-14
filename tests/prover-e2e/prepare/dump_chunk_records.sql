-- Create a file with INSERT statements for the specific records
\o 00102_import_chunks.sql
\t on
\a
-- Write header comment
SELECT '-- +goose Up';
SELECT '-- +goose StatementBegin';
SELECT '';

SELECT
  'INSERT INTO chunk (' ||
    'index, hash, start_block_number, start_block_hash, end_block_number, end_block_hash, ' ||
    'start_block_time, total_l1_messages_popped_before, total_l1_messages_popped_in_chunk, ' ||
    'prev_l1_message_queue_hash, post_l1_message_queue_hash, parent_chunk_hash, state_root, ' ||
    'parent_chunk_state_root, withdraw_root, codec_version, enable_compress, ' ||
    'total_l2_tx_gas, total_l2_tx_num, total_l1_commit_calldata_size, total_l1_commit_gas, ' ||
    'created_at, updated_at' ||
  ') VALUES (' ||
    quote_literal(index) || ', ' ||
    quote_literal(hash) || ', ' ||
    quote_literal(start_block_number) || ', ' ||
    quote_literal(start_block_hash) || ', ' ||
    quote_literal(end_block_number) || ', ' ||
    quote_literal(end_block_hash) || ', ' ||
    quote_literal(start_block_time) || ', ' ||
    quote_literal(total_l1_messages_popped_before) || ', ' ||
    quote_literal(total_l1_messages_popped_in_chunk) || ', ' ||
    quote_literal(prev_l1_message_queue_hash) || ', ' ||
    quote_literal(post_l1_message_queue_hash) || ', ' ||
    quote_literal(parent_chunk_hash) || ', ' ||
    quote_literal(state_root) || ', ' ||
    quote_literal(parent_chunk_state_root) || ', ' ||
    quote_literal(withdraw_root) || ', ' ||
    quote_literal(codec_version) || ', ' ||
    quote_literal(enable_compress) || ', ' ||
    quote_literal(total_l2_tx_gas) || ', ' ||
    quote_literal(total_l2_tx_num) || ', ' ||
    quote_literal(total_l1_commit_calldata_size) || ', ' ||
    quote_literal(total_l1_commit_gas) || ', ' ||
    quote_literal(created_at) || ', ' ||
    quote_literal(updated_at) ||
  ');'
FROM chunk
ORDER BY index ASC;

-- Write footer
SELECT '';
SELECT '-- +goose StatementEnd';
SELECT '-- +goose Down';
SELECT '-- +goose StatementBegin';
SELECT 'DELETE FROM chunk;';
SELECT '-- +goose StatementEnd';

\t off
\a
\o