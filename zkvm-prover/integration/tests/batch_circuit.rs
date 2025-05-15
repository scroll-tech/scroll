use scroll_zkvm_integration::{
    ProverTester, prove_verify_multi, prove_verify_single,
    testers::{
        batch::{BatchProverTester, BatchTaskBuildingTester},
        chunk::MultiChunkProverTester,
    },
    utils::build_batch_task,
};

#[test]
fn test_execute() -> eyre::Result<()> {
    BatchProverTester::setup()?;

    let (_, app_config, exe_path) = BatchProverTester::load()?;
    let task = BatchProverTester::gen_proving_task()?;

    BatchProverTester::execute(app_config.clone(), &task, exe_path.clone())?;

    Ok(())
}

#[test]
fn test_e2e_execute() -> eyre::Result<()> {
    BatchProverTester::setup()?;

    let (_, app_config, exe_path) = BatchTaskBuildingTester::load()?;

    let task = BatchTaskBuildingTester::gen_proving_task()?;
    BatchTaskBuildingTester::execute(app_config.clone(), &task, exe_path.clone())?;

    Ok(())
}

#[test]
fn setup_prove_verify_single() -> eyre::Result<()> {
    BatchTaskBuildingTester::setup()?;

    prove_verify_single::<BatchTaskBuildingTester>(None)?;

    Ok(())
}

#[test]
fn e2e() -> eyre::Result<()> {
    BatchProverTester::setup()?;

    let outcome = prove_verify_multi::<MultiChunkProverTester>(None)?;

    let batch_task = build_batch_task(&outcome.tasks, &outcome.proofs, Default::default());
    prove_verify_single::<BatchProverTester>(Some(batch_task))?;

    Ok(())
}

#[cfg(feature = "euclidv2")]
#[test]
fn verify_batch_hash_invariant() -> eyre::Result<()> {
    use scroll_zkvm_integration::testers::chunk::gen_multi_tasks as gen_multi_chunk_tasks;
    BatchProverTester::setup()?;

    let outcome_1 = prove_verify_multi::<MultiChunkProverTester>(Some(
        gen_multi_chunk_tasks([vec![1], vec![2], vec![3, 4]])?.as_slice(),
    ))?;
    let outcome_2 = prove_verify_multi::<MultiChunkProverTester>(Some(
        gen_multi_chunk_tasks([vec![1, 2], vec![3], vec![4]])?.as_slice(),
    ))?;

    let batch_task_1 = build_batch_task(&outcome_1.tasks, &outcome_1.proofs, Default::default());
    let batch_task_2 = build_batch_task(&outcome_2.tasks, &outcome_2.proofs, Default::default());

    // verify the two task has the same blob bytes
    assert_eq!(
        batch_task_1
            .batch_header
            .must_v7_header()
            .blob_versioned_hash,
        batch_task_2
            .batch_header
            .must_v7_header()
            .blob_versioned_hash
    );

    Ok(())
}
