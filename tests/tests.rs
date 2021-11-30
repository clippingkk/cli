extern crate ck_cli;
use std::env;
use std::fs::File;
use std::io::prelude::*;
use serde_json;

#[test]
fn parse_en_file() {
	let mut current_dir = env::current_dir().unwrap();
	current_dir.push("fixtures");
	let mut src_en_file = current_dir.clone();
	src_en_file.push("clippings_en.txt");
	println!("{:?}", src_en_file);
	let mut f = File::open(src_en_file).unwrap();
	let mut result = String::new();
	f.read_to_string(&mut result).unwrap();

	let mut parsed_data = ck_cli::CKParser::do_parse(&result).unwrap();

	let mut result_en_file = current_dir.clone();
	result_en_file.push("clippings_en.result.json");
	let mut r = File::open(result_en_file).unwrap();
	let mut expected_json = String::new();
	r.read_to_string(&mut expected_json).unwrap();

	let parsed_json = serde_json::to_string(&parsed_data).unwrap();
	let mut expected_struct: Vec<ck_cli::CKParser::TClippingItem> = serde_json::from_str(&expected_json).unwrap();

	parsed_data.sort_by(|a, b| a.content.cmp(&b.content));
	expected_struct.sort_by(|a, b| a.content.cmp(&b.content));

	let ss = serde_json::to_string(&parsed_data).unwrap();
	let dd = serde_json::to_string(&expected_struct).unwrap();

	assert_eq!(ss, dd)
	// assert_eq!(parsed_data.eq(expected_struct), true)
	// assert_eq!(parsed_data, expected_struct)
	// assert_eq!(parsed_json, expected_json)
}
