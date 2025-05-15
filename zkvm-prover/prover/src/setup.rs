use std::{fs::read_to_string, path::Path};

use openvm_circuit::arch::instructions::exe::VmExe;
use openvm_native_recursion::halo2::utils::CacheHalo2ParamsReader;
use openvm_sdk::{
    DefaultStaticVerifierPvHandler, Sdk,
    commit::AppExecutionCommit,
    config::{AggConfig, AppConfig, SdkVmConfig},
    fs::{read_app_pk_from_file, read_exe_from_file},
    keygen::{AggProvingKey, AppProvingKey},
};
use openvm_stark_sdk::p3_baby_bear::BabyBear;

use crate::Error;

/// Alias for convenience.
pub type F = BabyBear;

/// Wrapper around [`openvm_sdk::fs::read_exe_from_file`].
pub fn read_app_exe<P: AsRef<Path>>(path: P) -> Result<VmExe<F>, Error> {
    read_exe_from_file(&path).map_err(|e| Error::Setup {
        path: path.as_ref().into(),
        src: e.to_string(),
    })
}

/// Wrapper around [`openvm_sdk::fs::read_app_pk_from_file`].
pub fn read_app_pk<P: AsRef<Path>>(path: P) -> Result<AppProvingKey<SdkVmConfig>, Error> {
    read_app_pk_from_file(&path).map_err(|e| Error::Setup {
        path: path.as_ref().into(),
        src: e.to_string(),
    })
}

/// Read and deserialize [`openvm_sdk::config::AppConfig`] from the given path to the TOML config.
pub fn read_app_config<P: AsRef<Path>>(path: P) -> Result<AppConfig<SdkVmConfig>, Error> {
    let toml_str = read_to_string(&path).map_err(|e| Error::Setup {
        path: path.as_ref().into(),
        src: e.to_string(),
    })?;

    toml::from_str(&toml_str).map_err(|e| Error::Setup {
        path: path.as_ref().into(),
        src: e.to_string(),
    })
}

/// Compute commitments to the app.
pub fn compute_commitments(
    app_exe: VmExe<F>,
    app_pk: AppProvingKey<SdkVmConfig>,
) -> Result<AppExecutionCommit<F>, Error> {
    let committed_exe = Sdk::new()
        .commit_app_exe(app_pk.app_fri_params(), app_exe)
        .map_err(|e| Error::Commit(e.to_string()))?;
    Ok(AppExecutionCommit::compute(
        &app_pk.app_vm_pk.vm_config,
        &committed_exe,
        &app_pk.leaf_committed_exe,
    ))
}

/// Generate STARK aggregation [proving key][`openvm_sdk::keygen::AggProvingKey`].
pub fn gen_agg_pk(params_dir: &str) -> Result<AggProvingKey, Error> {
    let halo2_params_reader = CacheHalo2ParamsReader::new(params_dir);
    Sdk::new()
        .agg_keygen(
            AggConfig::default(),
            &halo2_params_reader,
            &DefaultStaticVerifierPvHandler,
        )
        .map_err(|e| Error::Keygen(e.to_string()))
}
