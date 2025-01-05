use std::{collections::BTreeMap, rc::Rc};

use crate::types::ProverType;

use once_cell::sync::OnceCell;

use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};

use prover_darwin::BlockTrace as BlockTraceDarwin;
use prover_darwin_v2::BlockTrace as BlockTraceDarwinV2;

static mut PARAMS_MAP: OnceCell<Rc<BTreeMap<u32, ParamsKZG<Bn256>>>> = OnceCell::new();
pub enum BlockTraceType {
    LowVersion(Vec<BlockTraceDarwin>),
    HighVersion(Vec<BlockTraceDarwinV2>),
}

pub fn get_params_map_instance<'a, F>(load_params_func: F) -> &'a BTreeMap<u32, ParamsKZG<Bn256>>
where
    F: FnOnce() -> BTreeMap<u32, ParamsKZG<Bn256>>,
{
    unsafe {
        PARAMS_MAP.get_or_init(|| {
            let params_map = load_params_func();
            Rc::new(params_map)
        })
    }
}

pub fn get_degrees<F>(prover_types: &std::collections::HashSet<ProverType>, f: F) -> Vec<u32>
where
    F: FnMut(&ProverType) -> Vec<u32>,
{
    prover_types
        .iter()
        .flat_map(f)
        .collect::<std::collections::HashSet<u32>>()
        .into_iter()
        .collect()
}
