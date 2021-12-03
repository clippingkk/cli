use clap::Parser;
use std::fs::File;
use std::io;
use std::io::prelude::*;
use std::process;
use tokio;
use tokio::signal;
mod parser;

#[derive(Parser)]
#[clap(version = "2.0.0", author = "Annatar.He<annatar.he+ck.cli@gmail.com>")]
struct CommandOpts {
    #[clap(short = 'i', long, default_value = "")]
    input: String,
    #[clap(short = 'o', long, default_value = "")]
    output: String,
}

async fn main_fn() -> Result<(), Box<dyn std::error::Error>> {
    let opts: CommandOpts = CommandOpts::parse();

    let mut input_data: String = String::new();

    if !opts.input.eq("") {
        input_data = std::fs::read_to_string(opts.input)?;
    } else {
        io::stdin().read_to_string(&mut input_data)?;
    }

    let r = regex::Regex::new(r"\u{feff}").unwrap();

    let input = r.replace_all(&input_data, "");
    let result = parser::do_parse(&input.trim());

    if let Err(err) = result {
        eprintln!("{:?}", err);
        process::exit(255);
    }

    let out = serde_json::to_string_pretty(&result.unwrap()).unwrap();
    if opts.output.is_empty() {
        io::stdout().write(out.as_bytes()).unwrap();
    } else {
        let mut f = File::create(opts.output)?;
        f.write(out.as_bytes())?;
        f.flush()?;
    }

    process::exit(0);
}

async fn ctrlc_stop() -> Result<(), Box<dyn std::error::Error>> {
    signal::ctrl_c().await?;
    Ok(())
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let _ = tokio::join!(main_fn(), ctrlc_stop());
    Ok(())
}
