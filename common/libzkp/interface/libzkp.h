// BatchVerifier is used to:
// - Verify a batch proof
// - Verify a bundle proof
void init_batch_verifier(char* params_dir, char* assets_dir);

char verify_batch_proof(char* proof, char* fork_name);

char verify_bundle_proof(char* proof);

void init_chunk_verifier(char* params_dir, char* v3_assets_dir, char* v4_assets_dir);
char verify_chunk_proof(char* proof, char* fork_name);
