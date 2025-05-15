use sbv_primitives::B256;
use scroll_zkvm_integration::{
    ProverTester, prove_verify_multi, prove_verify_single, prove_verify_single_evm,
    testers::{
        batch::BatchProverTester,
        bundle::{BundleLocalTaskTester, BundleProverTester},
        chunk::{ChunkProverRv32Tester, ChunkProverTester, MultiChunkProverTester},
    },
    utils::{LastHeader, build_batch_task},
};
use scroll_zkvm_prover::{
    BatchProof, ChunkProof,
    task::{bundle::BundleProvingTask, chunk::ChunkProvingTask},
    utils::{read_json_deep, write_json},
};
use std::str::FromStr;

fn load_recent_batch_proofs() -> eyre::Result<BundleProvingTask> {
    let proof_path = glob::glob("../../.output/batch-tests-*/batch/proofs/batch-*.json")?
        .next()
        .unwrap()?;
    println!("proof_path: {:?}", proof_path);
    let batch_proof = read_json_deep::<_, BatchProof>(&proof_path)?;

    let task = BundleProvingTask {
        batch_proofs: vec![batch_proof],
        bundle_info: None,
        fork_name: if cfg!(feature = "euclidv2") {
            String::from("euclidv2")
        } else {
            String::from("euclidv1")
        },
    };
    Ok(task)
}

#[test]
fn setup_prove_verify() -> eyre::Result<()> {
    BundleProverTester::setup()?;

    let task = load_recent_batch_proofs()?;
    prove_verify_single_evm::<BundleProverTester>(Some(task))?;

    Ok(())
}

#[test]
fn setup_prove_verify_local_task() -> eyre::Result<()> {
    BundleLocalTaskTester::setup()?;
    prove_verify_single_evm::<BundleLocalTaskTester>(None)?;

    Ok(())
}

#[test]
fn verify_bundle_info_pi() {
    use scroll_zkvm_circuit_input_types::bundle::BundleInfo;

    let info = BundleInfo {
        chain_id: 534352,
        msg_queue_hash: Default::default(),
        num_batches: 12,
        prev_state_root: B256::from_str(
            "0x0090ecc1308e0033e8cfef3b6aabe1de0a93361a14075cf6246e002e62944fa3",
        )
        .unwrap(),
        prev_batch_hash: B256::from_str(
            "0x6f8315e6c702a9ea8f83fb46d2a4a8e4a01d46a5bf72de7fac179f373cf27d68",
        )
        .unwrap(),
        post_state_root: B256::from_str(
            "0x0e9c09b32fd71c248df1dbc2b8fcbf69839257296f447deb6a8f8f49b9e158e4",
        )
        .unwrap(),
        batch_hash: B256::from_str(
            "0x1655c7521aa3045f5267ff8c6b21f9ad42024f79369c447500fd04c1077c2ad5",
        )
        .unwrap(),
        withdraw_root: B256::from_str(
            "0x97f9728ad48ff896b4272abcecd9a6a46577c24fbf2504f5ed2c3178c857263a",
        )
        .unwrap(),
    };

    assert_eq!(
        info.pi_hash_euclidv1(),
        B256::from_str("0x5e49fc59ce02b42a2f693c738c582b36bd08e9cfe3acb8cee299216743869bd4")
            .unwrap()
    );
}

fn build_chunk_outcome() -> eyre::Result<(Vec<ChunkProvingTask>, Vec<ChunkProof>)> {
    let rv32_hybrid = false;
    if rv32_hybrid {
        let mut proofs = Vec::new();
        let tasks = MultiChunkProverTester::gen_multi_proving_tasks()?;
        for (idx, task) in tasks.iter().enumerate() {
            if idx % 2 == 0 {
                let outcome = prove_verify_single::<ChunkProverTester>(Some(task.clone()))?;
                proofs.push(outcome.proofs[0].clone());
            } else {
                let outcome = prove_verify_single::<ChunkProverRv32Tester>(Some(task.clone()))?;
                proofs.push(outcome.proofs[0].clone());
            }
        }
        Ok((tasks, proofs))
    } else {
        let outcome = prove_verify_multi::<MultiChunkProverTester>(None)?;
        Ok((outcome.tasks, outcome.proofs))
    }
}

#[test]
fn e2e() -> eyre::Result<()> {
    BundleProverTester::setup()?;

    let (chunk_tasks, chunk_proofs) = build_chunk_outcome()?;
    assert_eq!(chunk_tasks.len(), chunk_proofs.len());
    assert_eq!(chunk_tasks.len(), 3);

    // Construct batch tasks using chunk tasks and chunk proofs.
    let batch_task_1 =
        build_batch_task(&chunk_tasks[0..1], &chunk_proofs[0..1], Default::default());
    let batch_task_2 = build_batch_task(
        &chunk_tasks[1..],
        &chunk_proofs[1..],
        LastHeader::from(&batch_task_1.batch_header),
    );
    let batch_task_example = batch_task_1.clone();

    let outcome = prove_verify_multi::<BatchProverTester>(Some(&[batch_task_1, batch_task_2]))?;

    let fork_name = if cfg!(feature = "euclidv2") {
        String::from("euclidv2")
    } else {
        String::from("euclidv1")
    };
    // Construct bundle task using batch tasks and batch proofs.
    let bundle_task = BundleProvingTask {
        batch_proofs: outcome.proofs,
        bundle_info: None,
        fork_name: fork_name.clone(),
    };
    let (outcome, verifier, path_assets) =
        prove_verify_single_evm::<BundleProverTester>(Some(bundle_task.clone()))?;

    assert_eq!(outcome.proofs.len(), 1, "single bundle proof");

    let bundle_task_with_info = BundleProvingTask {
        batch_proofs: outcome.tasks[0].batch_proofs.clone(),
        bundle_info: Some(outcome.proofs[0].metadata.bundle_info.clone()),
        fork_name,
    };
    // collect batch and bundle task as data example
    write_json(path_assets.join("batch-task.json"), &batch_task_example)?;
    write_json(path_assets.join("bundle-task.json"), &bundle_task_with_info)?;

    // Verifier all above proofs with the verifier-only mode.
    let verifier = verifier.to_chunk_verifier();
    for proof in chunk_proofs.iter() {
        assert!(verifier.verify_proof(proof.as_proof()));
    }
    let verifier = verifier.to_batch_verifier();
    for proof in bundle_task.batch_proofs.iter() {
        assert!(verifier.verify_proof(proof.as_proof()));
    }
    #[cfg(not(feature = "euclidv2"))]
    let verifier = verifier.to_bundle_verifier_v1();
    #[cfg(feature = "euclidv2")]
    let verifier = verifier.to_bundle_verifier_v2();
    assert!(verifier.verify_proof_evm(&outcome.proofs[0].as_proof()));

    let expected_pi_hash = &outcome.proofs[0].metadata.bundle_pi_hash;
    let observed_instances = &outcome.proofs[0].as_proof().instances;

    for (i, (&expected, &observed)) in expected_pi_hash
        .iter()
        .zip(observed_instances.iter().skip(14).take(32))
        .enumerate()
    {
        assert_eq!(
            halo2curves_axiom::bn256::Fr::from(u64::from(expected)),
            observed,
            "pi inconsistent at index {i}: expected={expected}, observed={observed:?}"
        );
    }

    // Sanity check for pi of bundle hash, update the expected hash if block witness changed
    let pi_str = if cfg!(feature = "euclidv2") {
        "2028510c403837c6ed77660fd92814ba61d7b746e7268cc8dfc14d163d45e6bd"
    } else {
        "3cc70faf6b5a4bd565694a4c64de59befb735f4aac2a4b9e6a6fc2ee950b8a72"
    };
    // sanity check for pi of bundle hash, update the expected hash if block witness changed
    assert_eq!(
        alloy_primitives::hex::encode(expected_pi_hash),
        pi_str,
        "unexpected pi hash for e2e bundle info, block witness changed?"
    );

    Ok(())
}
