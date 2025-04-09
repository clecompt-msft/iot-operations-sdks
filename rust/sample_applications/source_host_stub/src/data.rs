use std::{future::Future, marker::Send, option::Option, pin::Pin};

use tokio::sync::mpsc::{UnboundedReceiver, error::TryRecvError};

use wasmtime::component::ResourceTable;
use wasmtime_wasi::{WasiCtx, WasiCtxBuilder, WasiView};

use crate::bindings::aio::connector::host;

pub struct Data {
    table: ResourceTable,
    ctx: WasiCtx,
    ch: UnboundedReceiver<host::Notification>,
}

impl Data {
    pub fn new(ch: UnboundedReceiver<host::Notification>) -> Self {
        let table = ResourceTable::new();
        let ctx = WasiCtxBuilder::new()
            .inherit_network()
            .inherit_stdout()
            .allow_ip_name_lookup(true)
            .build();

        Self { table, ctx, ch }
    }
}

impl WasiView for Data {
    fn table(&mut self) -> &mut ResourceTable {
        &mut self.table
    }

    fn ctx(&mut self) -> &mut WasiCtx {
        &mut self.ctx
    }
}

impl host::Host for Data {
    fn send<'life0, 'async_trait>(
        &'life0 mut self,
        kind: String,
        name: String,
        data: Vec<u8>,
    ) -> Pin<Box<dyn Future<Output = Result<(), String>> + Send + 'async_trait>>
    where
        'life0: 'async_trait,
        Self: 'async_trait,
    {
        if let Ok(body) = String::from_utf8(data) {
            println!("{}:{} - {}", kind, name, body);
        } else {
            println!("{}:{}", kind, name);
        }
        Box::pin(async move { Ok(()) })
    }

    fn poll<'life0, 'async_trait>(
        &'life0 mut self,
    ) -> Pin<Box<dyn Future<Output = Option<host::Notification>> + Send + 'async_trait>>
    where
        'life0: 'async_trait,
        Self: 'async_trait,
    {
        Box::pin(async move {
            match self.ch.try_recv() {
                Ok(n) => Some(n),
                Err(e) => match e {
                    TryRecvError::Disconnected => Some(host::Notification::Shutdown),
                    TryRecvError::Empty => None,
                },
            }
        })
    }
}
