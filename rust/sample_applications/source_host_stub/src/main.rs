use std::path::PathBuf;

use clap::Parser;

use tokio::sync::mpsc::{UnboundedSender, unbounded_channel};
use tokio::time::{Duration, sleep};

mod bindings {
    wasmtime::component::bindgen!({
        path: "../../../eng/wit",
        world: "source",
        async: true
    });
}
mod data;
mod runtime;

use bindings::aio::connector::host;

#[derive(Parser)]
struct Args {
    #[clap(value_name = "COMPONENT_PATH")]
    component: PathBuf,

    #[clap(value_name = "METHOD")]
    method: String,

    #[clap(value_name = "URL")]
    url: String,
}

#[tokio::main]
async fn main() -> wasmtime::Result<()> {
    let args = Args::parse();
    let mut rt = runtime::Runtime::new().await?;
    let (send, recv) = unbounded_channel::<host::Notification>();
    let mut inst = rt.instantiate(args.component, recv).await?;
    tokio::join!(inst.run(), run(send, args.method, args.url)).0
}

async fn run(ch: UnboundedSender<host::Notification>, method: String, url: String) {
    println!("send dataset-created");
    let _ = ch.send(host::Notification::DatasetCreated(host::Dataset {
        name: "test".to_owned(),
        poll_frequency_ms: 2000,
        additional_configuration: format!("{{\"method\":\"{}\",\"url\":\"{}\"}}", method, url)
            .to_owned(),
    }));

    sleep(Duration::from_secs(6)).await;

    println!("send dataset-removed");
    let _ = ch.send(host::Notification::DatasetRemoved("test".to_owned()));

    sleep(Duration::from_secs(4)).await;

    println!("done");
}
