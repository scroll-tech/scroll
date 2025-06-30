// Verifier is used to:
// - Verify a batch proof
// - Verify a bundle proof
// - Verify a chunk proof

#ifndef LIBZKP_H
#define LIBZKP_H

#include <stddef.h> // For size_t

// Initialize the verifier with configuration
void init_verifier(char* config);

// Initialize the l2geth with configuration
void init_l2geth(char* config);

// Verify proofs - returns non-zero for success, zero for failure
char verify_batch_proof(char* proof, char* fork_name);
char verify_bundle_proof(char* proof, char* fork_name);
char verify_chunk_proof(char* proof, char* fork_name);

// Dump verification key to file
void dump_vk(char* fork_name, char* file);

// The result struct to hold data from handling a proving task
typedef struct {
    char ok;
    char* universal_task;
    char* metadata;
    char expected_pi_hash[32];
} HandlingResult;

// Generate a universal task based on task type and input JSON
// Returns a struct containing task data, metadata, and expected proof hash
HandlingResult gen_universal_task(
    int task_type,
    char* task,
    char* fork_name,
    const unsigned char* expected_vk,
    size_t expected_vk_len
);

// Release memory allocated for a HandlingResult returned by gen_universal_task
void release_task_result(HandlingResult result);

// Generate a wrapped proof from the universal prover output and metadata
// Returns a JSON string containing the wrapped proof, or NULL on error
// The caller must call release_string on the returned pointer when done
char* gen_wrapped_proof(char* proof_json, char* metadata, char* vk, size_t vk_len);

// Release memory allocated for a string returned by gen_wrapped_proof
void release_string(char* string_ptr);

#endif /* LIBZKP_H */
