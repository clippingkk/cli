use clap::Parser;
use std::fs::File;
use std::io;
use std::io::prelude::*;
use std::process;
mod parser;

#[derive(Parser)]
#[clap(version = "2.0.0", author = "Annatar.He<annatar.he+ck.cli@gmail.com>")]
struct CommandOpts {
    #[clap(short = 'i', long, default_value = "")]
    input: String,
    #[clap(short = 'o', long, default_value = "")]
    output: String,
}

fn main() -> io::Result<()> {
    let opts: CommandOpts = CommandOpts::parse();

    let mut input_data: String = String::new();

    if !opts.input.eq("") {
        let mut f = File::open(opts.input)?;
        f.read_to_string(&mut input_data)?;
    } else {
        io::stdin().read_to_string(&mut input_data)?;
    }

    let r = regex::Regex::new(r"\u{feff}").unwrap();

    let input = r.replace_all(&input_data, "");

    // TODO: remove BOM from file
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

    Ok(())
}
