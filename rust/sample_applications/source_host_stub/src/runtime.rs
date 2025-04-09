use std::path::PathBuf;

use tokio::sync::mpsc::UnboundedReceiver;

use wasmtime::component::{Component, Linker};
use wasmtime::{Config, Engine, Store};

use crate::bindings::Source;
use crate::bindings::aio::connector::host;
use crate::data::Data;

pub struct Runtime {
    pub engine: Engine,
    pub linker: Linker<Data>,
}

impl Runtime {
    pub async fn new() -> wasmtime::Result<Self> {
        let mut config = Config::default();
        config.async_support(true);

        let engine = Engine::new(&config)?;
        let mut linker = Linker::new(&engine);
        wasmtime_wasi::add_to_linker_async(&mut linker)?;
        host::add_to_linker(&mut linker, |host| host)?;

        Ok(Self { engine, linker })
    }

    pub async fn instantiate(
        &mut self,
        path: PathBuf,
        ch: UnboundedReceiver<host::Notification>,
    ) -> wasmtime::Result<Instance> {
        let component = Component::from_file(&self.engine, path)?;
        let mut store = Store::new(&self.engine, Data::new(ch));
        let source = Source::instantiate_async(&mut store, &component, &self.linker).await?;
        Ok(Instance { store, source })
    }
}

pub struct Instance {
    store: Store<Data>,
    source: Source,
}

impl Instance {
    pub async fn run(&mut self) -> wasmtime::Result<()> {
        if let Err(e) = self.source.call_run(&mut self.store).await? {
            println!("{}", e);
        }

        Ok(())
    }
}
