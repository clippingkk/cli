extern crate ck_cli;
use std::env;
use std::fs::File;
use std::io::prelude::*;

use criterion::{black_box, criterion_group, criterion_main, Criterion};

fn pase_file(file_content: String) -> Vec<ck_cli::CKParser::TClippingItem> {
    let mut parsed_data = ck_cli::CKParser::do_parse(&file_content).unwrap();
    parsed_data
}

fn criterion_benchmark(c: &mut Criterion) {
    let file_name = "clippings_en";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();
    let file_content = result.repeat(black_box(1 << 10));
    c.bench_function("parse_file", |b| b.iter(|| pase_file(file_content.clone())));
}

criterion_group!(benches, criterion_benchmark);
criterion_main!(benches);
