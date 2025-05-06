use alloy_primitives::B256;
pub mod batch;
pub mod bundle;
pub mod chunk;

/// Defines behaviour to be implemented by types representing the public-input values of a circuit.
pub trait PublicInputs {
    /// Keccak-256 digest of the public inputs. The public-input hash are revealed as public values
    /// via [`openvm::io::reveal`].
    fn pi_hash(&self) -> B256;

    /// Validation logic between public inputs of two contiguous instances.
    fn validate(&self, prev_pi: &Self);
}

#[derive(
    Default,
    Debug,
    Copy,
    Clone,
    PartialEq,
    Eq,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub enum ForkName {
    #[default]
    EuclidV1,
    EuclidV2,
}

impl From<&ArchivedForkName> for ForkName {
    fn from(archived: &ArchivedForkName) -> Self {
        match archived {
            ArchivedForkName::EuclidV1 => ForkName::EuclidV1,
            ArchivedForkName::EuclidV2 => ForkName::EuclidV2,
        }
    }
}

impl From<Option<&str>> for ForkName {
    fn from(value: Option<&str>) -> Self {
        match value {
            None => Default::default(),
            Some("euclidv1") => ForkName::EuclidV1,
            Some("euclidv2") => ForkName::EuclidV2,
            Some(s) => unreachable!("hardfork not accepted: {s}"),
        }
    }
}

impl From<&str> for ForkName {
    fn from(value: &str) -> Self {
        match value {
            "euclidv1" => ForkName::EuclidV1,
            "euclidv2" => ForkName::EuclidV2,
            s => unreachable!("hardfork not accepted: {s}"),
        }
    }
}

/// helper trait to extend PublicInputs
pub trait MultiVersionPublicInputs {
    fn pi_hash_by_fork(&self, fork_name: ForkName) -> B256;
    fn validate(&self, prev_pi: &Self, fork_name: ForkName);
}

impl<T: MultiVersionPublicInputs> PublicInputs for (T, ForkName) {
    fn pi_hash(&self) -> B256 {
        self.0.pi_hash_by_fork(self.1)
    }

    fn validate(&self, prev_pi: &Self) {
        assert_eq!(self.1, prev_pi.1);
        self.0.validate(&prev_pi.0, self.1)
    }
}
