mod darwin;
mod darwin_v2;

use super::geth_client::GethClient;
use crate::{
    config::{AssetsDirEnvConfig, Config},
    types::{ProverType, Task, TaskType},
    utils::get_task_types,
};
use anyhow::{bail, Result};
use darwin::DarwinHandler;
use darwin_v2::DarwinV2Handler;
use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};
use std::{cell::RefCell, collections::BTreeMap, rc::Rc};

type HardForkName = String;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

pub trait CircuitsHandler {
    fn get_vk(&self, task_type: TaskType) -> Option<Vec<u8>>;

    fn get_proof_data(&self, task_type: TaskType, task: &Task) -> Result<String>;
}

pub struct CircuitsHandlerProvider<'a, 'b> {
    prover_type: ProverType,
    config: &'a Config,
    geth_client: Option<Rc<RefCell<GethClient>>>,
    params_map: &'b BTreeMap<u32, ParamsKZG<Bn256>>,
    current_fork_name: Option<HardForkName>,
    current_circuit: Option<Rc<Box<dyn CircuitsHandler + 'b>>>,
}

impl<'a, 'b> CircuitsHandlerProvider<'a, 'b> {
    fn handler_builder(&self) -> Result<Box<dyn CircuitsHandler + 'b>> {
        log::info!(
            "now init zk circuits handler, hard_fork_name: {}",
            &self.config.low_version_circuit.hard_fork_name
        );
        AssetsDirEnvConfig::enable_first();
        DarwinHandler::new(
            self.prover_type,
            self.params_map,
            &self.config.low_version_circuit.assets_path,
            self.geth_client.clone(),
        )
        .map(move |handler| Box::new(handler) as Box<dyn CircuitsHandler + 'b>)
    }

    fn next_handler_builder(&self) -> Result<Box<dyn CircuitsHandler + 'b>> {
        log::info!(
            "now init zk circuits handler, hard_fork_name: {}",
            &self.config.high_version_circuit.hard_fork_name
        );
        AssetsDirEnvConfig::enable_second();
        DarwinV2Handler::new(
            self.prover_type,
            self.params_map,
            &self.config.high_version_circuit.assets_path,
            self.geth_client.clone(),
        )
        .map(move |handler| Box::new(handler) as Box<dyn CircuitsHandler + 'b>)
    }

    pub fn new(
        prover_type: ProverType,
        config: &'a Config,
        params_map: &'b BTreeMap<u32, ParamsKZG<Bn256>>,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Result<Self> {
        let provider = CircuitsHandlerProvider {
            prover_type,
            config,
            geth_client,
            params_map,
            current_fork_name: None,
            current_circuit: None,
        };

        Ok(provider)
    }

    pub fn get_circuits_handler(
        &mut self,
        hard_fork_name: &String,
    ) -> Result<Rc<Box<dyn CircuitsHandler + 'b>>> {
        match &self.current_fork_name {
            Some(fork_name) if fork_name == hard_fork_name => {
                log::info!("get circuits handler from cache");
                if let Some(handler) = &self.current_circuit {
                    Ok(handler.clone())
                } else {
                    bail!("missing cached handler, there must be something wrong.")
                }
            }
            _ => {
                log::info!(
                    "failed to get circuits handler from cache, create a new one: {hard_fork_name}"
                );
                if ![
                    &self.config.high_version_circuit.hard_fork_name,
                    &self.config.low_version_circuit.hard_fork_name,
                ]
                .contains(&hard_fork_name)
                {
                    bail!("missing builder, there must be something wrong.")
                }

                log::info!("building circuits handler for {hard_fork_name}");
                self.current_fork_name = Some(hard_fork_name.clone());
                let handler = if hard_fork_name == &self.config.high_version_circuit.hard_fork_name
                {
                    self.handler_builder()
                        .expect("failed to build circuits handler")
                } else {
                    self.next_handler_builder()
                        .expect("failed to build circuits handler")
                };
                let rc_handler = Rc::new(handler);
                self.current_circuit = Some(rc_handler.clone());
                Ok(rc_handler)
            }
        }
    }

    pub fn init_vks(
        &self,
        prover_type: ProverType,
        config: &'a Config,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Vec<String> {
        [
            &config.low_version_circuit.hard_fork_name,
            &config.high_version_circuit.hard_fork_name,
        ]
        .into_iter()
        .flat_map(|hard_fork_name| {
            let handler = if hard_fork_name == &self.config.high_version_circuit.hard_fork_name {
                self.handler_builder()
                    .expect("failed to build circuits handler")
            } else {
                self.next_handler_builder()
                    .expect("failed to build circuits handler")
            };

            get_task_types(prover_type)
                .into_iter()
                .map(|task_type| {
                    let vk = handler
                        .get_vk(task_type)
                        .map_or("".to_string(), utils::encode_vk);
                    log::info!(
                        "vk for {hard_fork_name}, is {vk}, task_type: {:?}",
                        task_type
                    );
                    vk
                })
                .filter(|vk| !vk.is_empty())
                .collect::<Vec<String>>()
        })
        .collect::<Vec<String>>()
    }
}
