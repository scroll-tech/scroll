use eyre::Result;
use openvm_circuit::system::memory::merkle::public_values::UserPublicValuesProof;
use openvm_sdk::{DeferralInput, SC};
use openvm_stark_sdk::openvm_stark_backend::{codec::Decode, proof::Proof};
use openvm_verify_stark_host::{
    deferral::DeferralMerkleProofs,
    vk::{VerificationBaseline, VmStarkVerifyingKey},
    VmStarkProof,
};
use scroll_zkvm_types::proof::StarkProof;
use std::io::Cursor;

/// Compute deferred STARK verification data required by OpenVM v2+ aggregation circuits.
///
/// For batch/bundle proving the guest reads `input_commits` from stdin and the host must
/// additionally supply `DeferralInput`s and `DeferralState`s to the SDK prover. This function
/// derives all three from the child STARK proofs, the child aggregation VK and the parent's
/// deferral cached commit.
pub fn compute_deferral_data(
    child_agg_vk: &openvm_stark_sdk::openvm_stark_backend::keygen::types::MultiStarkVerifyingKey<
        SC,
    >,
    parent_deferral_cached_commit: openvm_continuations::CommitBytes,
    proofs: &[&StarkProof],
) -> Result<(
    Vec<[u8; 32]>,
    Vec<DeferralInput>,
    Vec<openvm_circuit::arch::deferral::DeferralState>,
)> {
    let mvk = (*child_agg_vk).clone();

    let (vm_proofs, baselines): (Vec<VmStarkProof>, Vec<VerificationBaseline>) = proofs
        .iter()
        .map(|p| decode_stark_proof(p))
        .collect::<Result<Vec<_>>>()?
        .into_iter()
        .unzip();

    let baseline = baselines
        .first()
        .cloned()
        .ok_or_else(|| eyre::eyre!("no child proofs to compute deferral data"))?;
    for (i, b) in baselines.iter().enumerate().skip(1) {
        if b.app_exe_commit != baseline.app_exe_commit || b.app_vk_commit != baseline.app_vk_commit
        {
            eyre::bail!(
                "child proof {i} has a different verification baseline; all child proofs must use the same app exe/vk commitment"
            );
        }
    }

    let vk = VmStarkVerifyingKey { mvk, baseline };

    let cached_commit: openvm_stark_sdk::config::baby_bear_poseidon2::Digest =
        parent_deferral_cached_commit.into();

    let raw_results = openvm_verify_stark_circuit::extension::get_raw_deferral_results(
        &vk,
        &vm_proofs,
        cached_commit,
    )
    .map_err(|e| eyre::eyre!("get_raw_deferral_results failed: {e}"))?;

    let input_commits: Vec<[u8; 32]> = raw_results
        .iter()
        .map(|r| {
            r.input
                .as_slice()
                .try_into()
                .expect("input commit must be 32 bytes")
        })
        .collect();

    let deferral_inputs = vec![DeferralInput::from_inputs(&vm_proofs)];

    let deferral_state = openvm_verify_stark_circuit::extension::get_deferral_state(
        &vk,
        &vm_proofs,
        cached_commit,
        0,
    )
    .map_err(|e| eyre::eyre!("get_deferral_state failed: {e}"))?;

    Ok((input_commits, deferral_inputs, vec![deferral_state]))
}

fn decode_stark_proof(proof: &StarkProof) -> Result<(VmStarkProof, VerificationBaseline)> {
    let inner = Proof::<SC>::decode_from_bytes(&proof.proof)
        .map_err(|e| eyre::eyre!("decode proof failed: {e}"))?;
    let user_pvs_proof =
        UserPublicValuesProof::decode::<SC, _>(&mut Cursor::new(&proof.user_pvs_proof))
            .map_err(|e| eyre::eyre!("decode user_pvs_proof failed: {e}"))?;
    let deferral_merkle_proofs = if proof.deferral_merkle_proofs.is_empty() {
        None
    } else {
        Some(
            DeferralMerkleProofs::decode(&mut Cursor::new(&proof.deferral_merkle_proofs))
                .map_err(|e| eyre::eyre!("decode deferral_merkle_proofs failed: {e}"))?,
        )
    };
    let baseline = if proof.baseline.is_empty() {
        eyre::bail!("stark proof missing verification baseline");
    } else {
        serde_json::from_slice(&proof.baseline)
            .map_err(|e| eyre::eyre!("decode baseline failed: {e}"))?
    };
    Ok((
        VmStarkProof {
            inner,
            user_pvs_proof,
            deferral_merkle_proofs,
        },
        baseline,
    ))
}
