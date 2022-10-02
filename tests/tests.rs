extern crate ck_cli;
use serde_json;
use std::{env, fs};
use std::fs::File;
use std::io::prelude::*;

#[test]
fn parse_en_file() {
    let file_name = "clippings_en";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();

    let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

    let mut result_en_file = current_dir.clone();
    result_en_file.push(format!("{}.result.json", file_name));
    let mut r = File::open(result_en_file).unwrap();
    let mut expected_json = String::new();
    r.read_to_string(&mut expected_json).unwrap();

    let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> =
        serde_json::from_str(&expected_json).unwrap();

    parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
    expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

    let ss = serde_json::to_string(&parsed_data).unwrap();
    let dd = serde_json::to_string(&expected_struct).unwrap();

    assert_eq!(ss, dd)
}

#[test]
fn parse_other_file() {
    let file_name = "clippings_other";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();

    let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

    let mut result_en_file = current_dir.clone();
    result_en_file.push(format!("{}.result.json", file_name));
    let mut r = File::open(result_en_file).unwrap();
    let mut expected_json = String::new();
    r.read_to_string(&mut expected_json).unwrap();

    let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> =
        serde_json::from_str(&expected_json).unwrap();

    parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
    expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

    let ss = serde_json::to_string(&parsed_data).unwrap();
    let dd = serde_json::to_string(&expected_struct).unwrap();

    assert_eq!(ss, dd)
}
#[test]
fn parse_ric_file() {
    let file_name = "clippings_ric";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();

    let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

    let mut result_en_file = current_dir.clone();
    result_en_file.push(format!("{}.result.json", file_name));
    let mut r = File::open(result_en_file).unwrap();
    let mut expected_json = String::new();
    r.read_to_string(&mut expected_json).unwrap();

    let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> =
        serde_json::from_str(&expected_json).unwrap();

    parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
    expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

    let ss = serde_json::to_string(&parsed_data).unwrap();
    let dd = serde_json::to_string(&expected_struct).unwrap();

    assert_eq!(ss, dd)
}
#[test]
fn parse_zh_file() {
    let file_name = "clippings_zh";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();

    let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

    let mut result_en_file = current_dir.clone();
    result_en_file.push(format!("{}.result.json", file_name));
    let mut r = File::open(result_en_file).unwrap();
    let mut expected_json = String::new();
    r.read_to_string(&mut expected_json).unwrap();

    let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> =
        serde_json::from_str(&expected_json).unwrap();

    parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
    expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

    let ss = serde_json::to_string(&parsed_data).unwrap();
    let dd = serde_json::to_string(&expected_struct).unwrap();

    assert_eq!(ss, dd)
}

#[test]
fn parse_rare_file() {
    let file_name = "clippings_rare";
    let mut current_dir = env::current_dir().unwrap();
    current_dir.push("fixtures");
    let mut src_en_file = current_dir.clone();
    src_en_file.push(format!("{}.txt", file_name));
    let mut f = File::open(src_en_file).unwrap();
    let mut result = String::new();
    f.read_to_string(&mut result).unwrap();

    let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

    let mut result_en_file = current_dir.clone();
    result_en_file.push(format!("{}.result.json", file_name));
    let mut r = File::open(result_en_file).unwrap();
    let mut expected_json = String::new();
    r.read_to_string(&mut expected_json).unwrap();

    let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> =
        serde_json::from_str(&expected_json).unwrap();

    parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
    expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

    let ss = serde_json::to_string(&parsed_data).unwrap();
    let dd = serde_json::to_string(&expected_struct).unwrap();

    assert_eq!(ss, dd)
}
