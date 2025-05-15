use openvm_native_recursion::halo2::RawEvmProof;
use revm::{
    Context, Evm, Handler, InMemoryDB,
    primitives::{ExecutionResult, Output, TransactTo, TxEnv, TxKind, specification::CancunSpec},
};

// Re-export from snark_verifier_sdk.
pub use snark_verifier_sdk::{
    evm::gen_evm_verifier_shplonk as gen_evm_verifier,
    halo2::aggregation as halo2_aggregation,
    snark_verifier::halo2_base::halo2_proofs::{
        SerdeFormat,
        halo2curves::bn256::{Fr, G1Affine},
        plonk::{Circuit, VerifyingKey},
    },
};

/// Serialize vk, code extracted from legacy prover.
pub fn serialize_vk(vk: &VerifyingKey<G1Affine>) -> Vec<u8> {
    let mut result = Vec::<u8>::new();
    vk.write(&mut result, SerdeFormat::Processed).unwrap();
    result
}

/// Deserialize vk, code extracted from legacy prover.
///
/// Panics if the deserialization fails.
pub fn deserialize_vk<C: Circuit<Fr, Params = ()>>(raw_vk: &[u8]) -> VerifyingKey<G1Affine> {
    VerifyingKey::<G1Affine>::read::<_, C>(
        &mut std::io::Cursor::new(raw_vk),
        SerdeFormat::Processed,
        (),
    )
    .unwrap_or_else(|_| panic!("failed to deserialize vk with len {}", raw_vk.len()))
}

/// Deploys the [`EvmVerifier`] contract and simulates an on-chain verification of the
/// [`EvmProof`].
///
/// This approach essentially simulates 2 txs:
/// - Deploy [`EvmVerifier`].
/// - Verify [`EvmProof`] encoded as calldata.
pub fn verify_evm_proof(evm_verifier: &[u8], evm_proof: &RawEvmProof) -> Result<u64, String> {
    let calldata =
        snark_verifier_sdk::evm::encode_calldata(&[evm_proof.instances.clone()], &evm_proof.proof);
    deploy_and_call(evm_verifier.to_vec(), calldata)
}

fn deploy_and_call(deployment_code: Vec<u8>, calldata: Vec<u8>) -> Result<u64, String> {
    let mut evm = Evm::new(
        Context::new_with_db(InMemoryDB::default()),
        Handler::mainnet::<CancunSpec>(),
    );

    *evm.tx_mut() = TxEnv {
        gas_limit: u64::MAX,
        transact_to: TxKind::Create,
        data: deployment_code.into(),
        ..Default::default()
    };

    let result = evm.transact_commit().unwrap();
    let contract = match result {
        ExecutionResult::Success {
            output: Output::Create(_, Some(contract)),
            ..
        } => contract,
        ExecutionResult::Revert { gas_used, output } => {
            return Err(format!(
                "Contract deployment transaction reverts with gas_used {gas_used} and output {:#x}",
                output
            ));
        }
        ExecutionResult::Halt { reason, gas_used } => {
            return Err(format!(
                "Contract deployment transaction halts unexpectedly with gas_used {gas_used} and reason {:?}",
                reason
            ));
        }
        _ => unreachable!(),
    };

    *evm.tx_mut() = TxEnv {
        gas_limit: u64::MAX,
        transact_to: TransactTo::Call(contract),
        data: calldata.into(),
        ..Default::default()
    };

    let result = evm.transact_commit().unwrap();
    match result {
        ExecutionResult::Success { gas_used, .. } => Ok(gas_used),
        ExecutionResult::Revert { gas_used, output } => Err(format!(
            "Contract call transaction reverts with gas_used {gas_used} and output {:#x}",
            output
        )),
        ExecutionResult::Halt { reason, gas_used } => Err(format!(
            "Contract call transaction halts unexpectedly with gas_used {gas_used} and reason {:?}",
            reason
        )),
    }
}

#[test]
fn test_verify_evm_proof() -> eyre::Result<()> {
    use std::path::Path;

    use scroll_zkvm_prover::{BundleProof, utils::read_json_deep};

    const PATH_TESTDATA: &str = "./testdata";

    let evm_proof = read_json_deep::<_, BundleProof>(
        Path::new(PATH_TESTDATA)
            .join("proofs")
            .join("bundle-proof-phase2.json"),
    )?;

    let evm_verifier: Vec<u8> =
        scroll_zkvm_prover::utils::read(Path::new(PATH_TESTDATA).join("verifier.bin"))?;

    let gas_cost = verify_evm_proof(&evm_verifier, &evm_proof.as_proof())
        .map_err(|e| eyre::eyre!("evm-proof verification failed: {e}"))?;

    println!("evm-verify gas cost = {gas_cost}");

    Ok(())
}
