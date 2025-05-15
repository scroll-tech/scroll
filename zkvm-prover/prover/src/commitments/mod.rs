mod batch_leaf_commit;
mod batch_exe_commit;
mod chunk_exe_commit;
mod chunk_exe_rv32_commit;
mod chunk_leaf_commit;
mod bundle_leaf_commit;
mod bundle_exe_commit;
mod bundle_exe_euclidv1_commit;

pub mod batch {
    use super::batch_exe_commit;
    use super::batch_leaf_commit;

    pub const EXE_COMMIT: [u32; 8] = batch_exe_commit::COMMIT;
    pub const LEAF_COMMIT: [u32; 8] = batch_leaf_commit::COMMIT;
}

pub mod bundle {
    use super::bundle_exe_commit;
    use super::bundle_leaf_commit;

    pub const EXE_COMMIT: [u32; 8] = bundle_exe_commit::COMMIT;
    pub const LEAF_COMMIT: [u32; 8] = bundle_leaf_commit::COMMIT;
}

pub mod bundle_euclidv1 {

    use super::bundle_exe_euclidv1_commit;
    use super::bundle_leaf_commit;

    pub const EXE_COMMIT: [u32; 8] = bundle_exe_euclidv1_commit::COMMIT;
    pub const LEAF_COMMIT: [u32; 8] = bundle_leaf_commit::COMMIT;
}

pub mod chunk {
    use super::chunk_exe_commit;
    use super::chunk_leaf_commit;

    pub const EXE_COMMIT: [u32; 8] = chunk_exe_commit::COMMIT;
    pub const LEAF_COMMIT: [u32; 8] = chunk_leaf_commit::COMMIT;
}

pub mod chunk_rv32 {
    use super::chunk_exe_rv32_commit;
    use super::chunk_leaf_commit;

    pub const EXE_COMMIT: [u32; 8] = chunk_exe_rv32_commit::COMMIT;
    pub const LEAF_COMMIT: [u32; 8] = chunk_leaf_commit::COMMIT;
}
