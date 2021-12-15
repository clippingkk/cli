#[macro_use]
extern crate colour;

use crate::config::ensure_toml_config;
use clap::{AppSettings, Parser, Subcommand};
use std::io;
use std::io::prelude::*;
use std::process;
use tokio;
use tokio::signal;
mod config;
mod constants;
mod graphql;
mod http;
mod parser;

#[derive(Subcommand)]
enum Commands {
    // #[clap(setting(AppSettings::ArgRequiredElseHelp))]
    Login {},
    #[clap(setting(AppSettings::ArgRequiredElseHelp))]
    Parse {
        #[clap(short = 'i', long, default_value = "")]
        input: String,
        #[clap(short = 'o', long, default_value = "")]
        output: String,
    },
}

#[derive(Parser)]
#[clap(name = "ck-cli")]
#[clap(version = "2.0.0", author = "Annatar.He<annatar.he+ck.cli@gmail.com>")]
struct CliCommands {
    #[clap(short = 'c', long, default_value = "")]
    config: String,
    #[clap(subcommand)]
    command: Commands,
}

async fn main_fn() -> Result<(), Box<dyn std::error::Error>> {
    let args = CliCommands::parse();
    let ck_config = ensure_toml_config(&args.config)?;

    match &args.command {
        Commands::Login {} => {
            // TODO: interactive
            // 1: phone number / email
            // 2: image verification
            // 3: sms code check
            // 4: receive auth response
            // 5: save to local config
            blue_ln!(" 💪  working on it")
        }
        Commands::Parse {
            input,
            output,
        } => {
            let mut input_data: String = String::new();

            if !input.eq("") {
                input_data = std::fs::read_to_string(input)?;
            } else {
                io::stdin().read_to_string(&mut input_data)?;
            }

            let r = regex::Regex::new(r"\u{feff}").unwrap();

            let input = r.replace_all(&input_data, "");
            let result = parser::do_parse(&input.trim());

            if let Err(err) = result {
                e_red_ln!(" ❌ {:?}", err);
                return Err(err);
            }

            let result_obj = result.unwrap();
            let out = serde_json::to_string_pretty(&result_obj).unwrap();
            if output.is_empty() {
                io::stdout().write(out.as_bytes())?;
            } else if output.starts_with("http") {
                http::sync_to_server(&output, &ck_config.http, &result_obj).await?;
            } else {
                std::fs::write(output, out)?;
            }
        }
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
