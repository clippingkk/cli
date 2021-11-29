use clap::{AppSettings, Parser};
use std::io;
use std::io::prelude::*;
use std::fs::File;
mod parser;

#[derive(Parser)]
#[clap(version = "2.0.0", author = "Annatar.He<annatar.he+ck.cli@gmail.com>")]
struct CommandOpts {
    #[clap(short='i', long, default_value="")]
    input: String,
    #[clap(short='o', long, default_value="")]
    output: String,
}

fn main() -> io::Result<()>{
    let opts: CommandOpts = CommandOpts::parse();

    let mut input_data: String = String::new();

    if !opts.input.eq("") {
        let mut f = File::open(opts.input)?;
        f.read_to_string(&mut input_data)?;
    } else {
        io::stdin().read_to_string(&mut input_data)?;
    }

    // TODO: remove BOM from file
    let result = parser::do_parse(&input_data);

    match result {
        Ok(res) => println!("{:?}", res),
        Err(err) => println!("{}", err),
    }

    Ok(())
}
