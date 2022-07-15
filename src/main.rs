#[macro_use]
extern crate colour;

use crate::config::ensure_toml_config;
use clap::{Parser, Subcommand};
use std::io;
use std::io::prelude::*;
use std::process;
use tokio;
use tokio::signal;
mod auth;
mod config;
mod constants;
mod graphql;
mod http;
mod parser;

#[derive(Subcommand)]
enum Commands {
    // #[clap(setting(AppSettings::ArgRequiredElseHelp))]
    Login {},
    // #[clap(setting(AppSettings::ArgRequiredElseHelp))]
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
    #[clap(short = 't', long, default_value = "")]
    token: String,
    #[clap(subcommand)]
    command: Commands,
}

async fn main_fn() -> Result<(), Box<dyn std::error::Error>> {
    let args = CliCommands::parse();
    let mut ck_config = ensure_toml_config(&args.config)?;
    let mut p = dirs::home_dir().unwrap();
    p.push(".ck-cli.toml");

    match &args.command {
        Commands::Login {} => {
            if args.token.is_empty() {
                e_red_ln!(" ❌ token not found \n visit https://clippingkk.annatarhe.com and login \n then navigate to your profile page and open `API Token` dialog. \n Copy it and paste to this cli.");
                process::exit(255);
            }
            ck_config = ck_config.update_token(&args.token)?;
            ck_config.save(&p.clone())?;

            green_ln!(" ✅ logged. you can synchronize your `My Clippings.txt` by run command \n $ ck-cli parse --input /path/to/My Clippings.txt --output http")
        }
        Commands::Parse { input, output } => {
            let ckc = ck_config.clone();
            let mut ckh = ckc.http.clone();
            if !args.token.is_empty() {
                ck_config = ck_config.update_token(&args.token)?;
                let nc = ck_config.save(&p.clone())?;
                ckh = nc.http;
            }

            let mut input_data: String = String::new();
            if !input.is_empty() {
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
                http::sync_to_server(&output, &ckh, &result_obj).await?;
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
