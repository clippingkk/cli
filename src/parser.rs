use std::vec::Vec;
use chrono::{DateTime,TimeZone, Utc};
use regex::Regex;

#[derive(Debug, Clone)]
pub struct TClippingItem {
	title: String,
	content: String,
	pageAt: String,
	createdAt: chrono::DateTime<Utc>,
}

pub enum KindleClippingFileLines {
	Title,
	Info,
	Content,
}
enum KindleClippingLanguage {
	Zh,
	En,
}


struct ParserLanguageConfig {
	location: regex::Regex,
	language: KindleClippingLanguage,
}

pub fn do_parse(input: &String) -> Result<Vec<TClippingItem>, String> {
	let separator = "========";
	let lines: Vec<&str> = input.split('\n').collect();
	let mut grouped: Vec<Vec<String>> = Vec::new();
	let mut temp: Vec<String> = Vec::new();


	let la_config: ParserLanguageConfig = if input.contains("Your Highlight on") {
	let a = Regex::new(r"\d").unwrap();
		ParserLanguageConfig { location: a, language: KindleClippingLanguage::En }
	} else {
	let a = Regex::new(r"\d").unwrap();
		ParserLanguageConfig { location: a, language: KindleClippingLanguage::Zh }
	};

	// TODO: reduce O(n^2) to O(n)
	for l in lines {
		if l.contains(separator) {
			grouped.push(temp.clone());
			temp.clear();
		} else {
			temp.push(String::from(l));
		}
	}

	let mut result_list: Vec<TClippingItem> = vec![];
	for row in grouped {
		let title = parse_title(&row[0]);

		let (location, dt) = parse_info(&row[1], &la_config.location)?;

		let item = TClippingItem {
			content: "".to_string(),
			title: title,
			pageAt: location,
			createdAt: dt,
		};
		result_list.push(item.clone());
	}

	// println!("{:?}", grouped);

	Ok(result_list)
}

fn parse_title(line: &String) -> String {
	let stop_worlds: Vec<&str> = vec!["(", "（"];

	let mut title = line.clone();

	for w in stop_worlds {
		title = title.split(w).collect();
	}

	title
}

fn parse_info(line: &String, location_regex: &Regex) -> Result<(String, chrono::DateTime<Utc>), String> {
	let ls: Vec<&str> = line.split("|").collect();
	let location_section = ls[0];
	let date_section = ls[ls.len() - 1];

	let location_result = "".to_string();
	let dt = chrono::offset::Utc::now();

	Ok((location_result, dt))
}

// const KindleDateTimeLayout {
// 	ENLayout = "Monday, January 2, 2006 3:4:5 PM"
// 	KindleDateTimeZHLayout = "2006-1-2 3:4:5"
// }
