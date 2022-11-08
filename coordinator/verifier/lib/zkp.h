init_prover(char *params_path, char *seed_path);
char* create_agg_proof(char *trace);
init_verifier(char *params_path, char *agg_vk_path);
char verify_agg_proof(char *proof);