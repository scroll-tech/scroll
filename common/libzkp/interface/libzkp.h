// BatchProver is used to:
// - Batch a list of chunk proofs
// - Bundle a list of batch proofs
void init_batch_prover(char* params_dir, char* assets_dir);
// BatchVerifier is used to:
// - Verify a batch proof
// - Verify a bundle proof
void init_batch_verifier(char* params_dir, char* assets_dir);

char* get_batch_vk();
char* check_chunk_proofs(char* chunk_proofs);
char* gen_batch_proof(char* chunk_hashes, char* chunk_proofs, char* batch_header);
char verify_batch_proof(char* proof, char* fork_name);

char* get_bundle_vk();
char* gen_bundle_proof(char* batch_proofs);
char verify_bundle_proof(char* proof);

void init_chunk_prover(char* params_dir, char* assets_dir);
void init_chunk_verifier(char* params_dir, char* assets_dir);
char* get_chunk_vk();
char* gen_chunk_proof(char* block_traces);
char verify_chunk_proof(char* proof, char* fork_name);

char* block_traces_to_chunk_info(char* block_traces);
void free_c_chars(char* ptr);