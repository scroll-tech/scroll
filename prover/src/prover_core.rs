use anyhow::{bail, Context, Error, Ok, Result};
use ethers_core::types::U64;

use std::{cell::RefCell, rc::Rc};

use crate::{
    config::Config,
    geth_client::GethClient,
    key_signer::KeySigner,
    types::{ProofFailureType, ProofStatus, ProofType},
    zk_circuits_handler::{CircuitsHandler, CircuitsHandlerProvider},
    types::{ProofDetail, Task},
};

pub struct Prover<'a> {
    config: &'a Config,
    key_signer: Rc<KeySigner>,
    circuits_handler_provider: RefCell<CircuitsHandlerProvider<'a>>,
    geth_client: Option<Rc<RefCell<GethClient>>>,
}

impl<'a> Prover<'a> {
    pub fn new(config: &'a Config) -> Result<Self> {
        let proof_type = config.proof_type;
        let keystore_path = &config.keystore_path;
        let keystore_password = &config.keystore_password;

        let key_signer = Rc::new(KeySigner::new(keystore_path, keystore_password)?);

        let geth_client = if config.proof_type == ProofType::Chunk {
            Some(Rc::new(RefCell::new(
                GethClient::new(
                    &config.prover_name,
                    &config.l2geth.as_ref().unwrap().endpoint,
                )
                .context("failed to create l2 geth_client")?,
            )))
        } else {
            None
        };

        let provider = CircuitsHandlerProvider::new(proof_type, config, geth_client.clone())
            .context("failed to create circuits handler provider")?;

        let prover = Prover {
            config,
            key_signer: Rc::clone(&key_signer),
            circuits_handler_provider: RefCell::new(provider),
            geth_client,
        };

        Ok(prover)
    }

    pub fn get_proof_type(&self) -> ProofType {
        self.config.proof_type
    }

    pub fn get_public_key(&self) -> String {
        self.key_signer.get_public_key()
    }

    pub fn prove_task(&self, task: &Task) -> Result<ProofDetail> {
        log::info!("[prover] start to prove_task, task id: {}", task.id);
        let handler: Rc<Box<dyn CircuitsHandler>> = self
            .circuits_handler_provider
            .borrow_mut()
            .get_circuits_handler(&task.hard_fork_name)
            .context("failed to get circuit handler")?;
        self.do_prove(task, handler)
    }

    fn do_prove(&self, task: &Task, handler: Rc<Box<dyn CircuitsHandler>>) -> Result<ProofDetail> {
        let mut proof_detail = ProofDetail {
            id: task.id.clone(),
            proof_type: task.task_type,
            ..Default::default()
        };

        proof_detail.proof_data = handler.get_proof_data(task.task_type, task)?;
        Ok(proof_detail)
    }

    fn get_latest_block_number_value(&self) -> Result<Option<U64>> {
        let number = self
            .geth_client
            .as_ref()
            .unwrap()
            .borrow_mut()
            .block_number()?;
        Ok(number.as_number())
    }
}
