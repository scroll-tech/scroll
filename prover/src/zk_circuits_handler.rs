mod darwin;
mod edison;

use super::geth_client::GethClient;
use crate::{
    config::{AssetsDirEnvConfig, Config},
    types::{ProverType, Task, TaskType},
    utils::get_task_types,
};
use anyhow::{bail, Result};
use darwin::DarwinHandler;
use edison::EdisonHandler;
use std::{cell::RefCell, collections::HashMap, rc::Rc};

type CircuitsVersion = String;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

pub trait CircuitsHandler {
    fn get_vk(&self, task_type: TaskType) -> Option<Vec<u8>>;

    fn get_proof_data(&self, task_type: TaskType, task: &Task) -> Result<String>;
}

type CircuitsHandlerBuilder = fn(
    prover_type: ProverType,
    config: &Config,
    geth_client: Option<Rc<RefCell<GethClient>>>,
) -> Result<Box<dyn CircuitsHandler>>;

pub struct CircuitsHandlerProvider<'a> {
    prover_type: ProverType,
    config: &'a Config,
    geth_client: Option<Rc<RefCell<GethClient>>>,
    circuits_handler_builder_map: HashMap<CircuitsVersion, CircuitsHandlerBuilder>,

    current_circuits_version: Option<CircuitsVersion>,
    current_circuit: Option<Rc<Box<dyn CircuitsHandler>>>,
}

impl<'a> CircuitsHandlerProvider<'a> {
    pub fn new(
        prover_type: ProverType,
        config: &'a Config,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Result<Self> {
        let mut m: HashMap<CircuitsVersion, CircuitsHandlerBuilder> = HashMap::new();

        fn handler_builder(
            prover_type: ProverType,
            config: &Config,
            geth_client: Option<Rc<RefCell<GethClient>>>,
        ) -> Result<Box<dyn CircuitsHandler>> {
            log::info!(
                "now init zk circuits handler, hard_fork_name: {}, version: {}",
                &config.low_version_circuit.hard_fork_name,
                &config.low_version_circuit.circuits_version,
            );
            AssetsDirEnvConfig::enable_first();
            DarwinHandler::new(
                prover_type,
                &config.low_version_circuit.params_path,
                &config.low_version_circuit.assets_path,
                geth_client,
            )
            .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }
        m.insert(
            config.low_version_circuit.circuits_version.clone(),
            handler_builder,
        );

        fn next_handler_builder(
            prover_type: ProverType,
            config: &Config,
            geth_client: Option<Rc<RefCell<GethClient>>>,
        ) -> Result<Box<dyn CircuitsHandler>> {
            log::info!(
                "now init zk circuits handler, hard_fork_name: {}, version: {}",
                &config.high_version_circuit.hard_fork_name,
                &config.high_version_circuit.circuits_version
            );
            AssetsDirEnvConfig::enable_second();
            EdisonHandler::new(
                prover_type,
                &config.high_version_circuit.params_path,
                &config.high_version_circuit.assets_path,
                geth_client,
            )
            .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }

        m.insert(
            config.high_version_circuit.circuits_version.clone(),
            next_handler_builder,
        );

        let provider = CircuitsHandlerProvider {
            prover_type,
            config,
            geth_client,
            circuits_handler_builder_map: m,
            current_circuits_version: None,
            current_circuit: None,
        };

        Ok(provider)
    }

    pub fn get_circuits_handler(
        &mut self,
        circuits_version: &String,
    ) -> Result<Rc<Box<dyn CircuitsHandler>>> {
        match &self.current_circuits_version {
            Some(version) if version == circuits_version => {
                log::info!("get circuits handler from cache");
                if let Some(handler) = &self.current_circuit {
                    Ok(handler.clone())
                } else {
                    log::error!("missing cached handler, there must be something wrong.");
                    bail!("missing cached handler, there must be something wrong.")
                }
            }
            _ => {
                log::info!(
                    "failed to get circuits handler from cache, create a new one: {circuits_version}"
                );
                if let Some(builder) = self.circuits_handler_builder_map.get(circuits_version) {
                    log::info!("building circuits handler for {circuits_version}");
                    let handler = builder(self.prover_type, self.config, self.geth_client.clone())
                        .expect("failed to build circuits handler");
                    self.current_circuits_version = Some(circuits_version.clone());
                    let rc_handler = Rc::new(handler);
                    self.current_circuit = Some(rc_handler.clone());
                    Ok(rc_handler)
                } else {
                    log::error!("missing builder, there must be something wrong.");
                    bail!("missing builder, there must be something wrong.")
                }
            }
        }
    }

    pub fn init_vks(
        &self,
        prover_type: ProverType,
        config: &'a Config,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Vec<String> {
        self.circuits_handler_builder_map
            .iter()
            .flat_map(|(hard_fork_name, build)| {
                let handler = build(prover_type, config, geth_client.clone())
                    .expect("failed to build circuits handler");

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
