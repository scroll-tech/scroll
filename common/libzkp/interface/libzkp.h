// BatchVerifier is used to:
// - Verify a batch proof
// - Verify a bundle proof
void init(char* config);

char verify_batch_proof(char* proof, char* fork_name, char* circuits_version);

char verify_bundle_proof(char* proof, char* fork_name, char* circuits_version);

char verify_chunk_proof(char* proof, char* fork_name, char* circuits_version);
