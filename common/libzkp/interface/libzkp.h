void init_batch_prover(char* params_dir);
void init_batch_verifier(char* params_dir, char* assets_dir);
char* gen_batch_proof(char* chunk_hashes, char* chunk_proofs);
char verify_batch_proof(char* proof);

void init_chunk_prover(char* params_dir);
void init_chunk_verifier(char* params_dir, char* assets_dir);
char* gen_chunk_proof(char* block_traces);
char verify_chunk_proof(char* proof);

char* block_traces_to_chunk_hash(char* block_traces);
