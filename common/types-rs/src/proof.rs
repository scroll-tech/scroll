/// Represents an openvm program commitments and public values.
#[derive(
    Clone,
    Debug,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct AggregationInput {
    /// Public values.
    pub public_values: Vec<u32>,
    /// Represent the commitment needed to verify a root proof
    pub commitment: ProgramCommitment,
}

/// Represent the commitment needed to verify a [`RootProof`].
#[derive(
    Clone,
    Debug,
    Default,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct ProgramCommitment {
    /// The commitment to the child program exe.
    pub exe: [u32; 8],
    /// The commitment to the child program leaf.
    pub leaf: [u32; 8],
}

impl ProgramCommitment {
    pub fn deserialize(commitment_bytes: &[u8]) -> Self {
        // TODO: temporary skip deserialize if no vk is provided
        if commitment_bytes.is_empty() {
            return Default::default();
        }

        let archived_data =
            rkyv::access::<ArchivedProgramCommitment, rkyv::rancor::BoxedError>(commitment_bytes)
                .unwrap();

        Self {
            exe: archived_data.exe.map(|u32_le| u32_le.to_native()),
            leaf: archived_data.leaf.map(|u32_le| u32_le.to_native()),
        }
    }

    pub fn serialize(&self) -> Vec<u8> {
        rkyv::to_bytes::<rkyv::rancor::BoxedError>(self)
            .map(|v| v.to_vec())
            .unwrap()
    }
}

impl From<&ArchivedProgramCommitment> for ProgramCommitment {
    fn from(archived: &ArchivedProgramCommitment) -> Self {
        Self {
            exe: archived.exe.map(|u32_le| u32_le.to_native()),
            leaf: archived.leaf.map(|u32_le| u32_le.to_native()),
        }
    }
}

/// Number of public-input values, i.e. [u32; N].
///
/// Note that the actual value for each u32 is a byte.
const NUM_PUBLIC_VALUES: usize = 32;

/// Verify a root proof. The real "proof" will be loaded from StdIn.
pub fn verify_proof(commitment: &ProgramCommitment, public_inputs: &[u32]) {
    // Sanity check for the number of public-input values.
    assert_eq!(public_inputs.len(), NUM_PUBLIC_VALUES);

    // Extend the public-input values by prepending the commitments to the root verifier's exe and
    // leaf.
    let mut extended_public_inputs = vec![];
    extended_public_inputs.extend(commitment.exe);
    extended_public_inputs.extend(commitment.leaf);
    extended_public_inputs.extend_from_slice(public_inputs);
    // Pass through kernel and verify against root verifier's ASM.
    exec_kernel(extended_public_inputs.as_ptr());
}

fn exec_kernel(_pi_ptr: *const u32) {
    // reserve x29, x30, x31 for kernel
    let mut _buf1: u32 = 0;
    let mut _buf2: u32 = 0;
    #[cfg(all(target_os = "zkvm", target_arch = "riscv32"))]
    unsafe {
        std::arch::asm!(
            include_str!("../../../build-guest/root_verifier.asm"),
            in("x29") _pi_ptr,
            inout("x30") _buf1,
            inout("x31") _buf2,
        )
    }
}
