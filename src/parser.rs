use chrono::{DateTime, TimeZone, Utc};
use regex::Regex;
use std::error::Error;
use std::vec::Vec;

#[derive(Debug, Clone)]
pub struct TClippingItem {
	title: String,
	content: String,
	pageAt: String,
	createdAt: chrono::NaiveDateTime,
}

pub enum KindleClippingFileLines {
	Title,
	Info,
	Content,
}
#[derive(Debug)]
enum KindleClippingLanguage {
	Zh,
	En,
}

struct ParserLanguageConfig {
	location: regex::Regex,
	language: KindleClippingLanguage,
}

pub fn do_parse(input: &str) -> Result<Vec<TClippingItem>, Box<dyn Error>> {
	let separator = "========";
	let lines: Vec<&str> = input.split('\n').collect();
	let mut grouped: Vec<Vec<String>> = Vec::new();
	let mut temp: Vec<String> = Vec::new();

	let la_config: ParserLanguageConfig = if input.contains("Your Highlight on") {
		let a = Regex::new(r"\d+(-?\d+)?").unwrap();
		ParserLanguageConfig {
			location: a,
			language: KindleClippingLanguage::En,
		}
	} else {
		let a = Regex::new(r"#?\d+(-?\d+)?").unwrap();
		ParserLanguageConfig {
			location: a,
			language: KindleClippingLanguage::Zh,
		}
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
	let chinese_regex = Regex::new(r"[\x{4E00}-\x{9FFF}|\x{3000}-\x{303F}]").unwrap();
	for row in grouped {
		let title = parse_title(&row[0]);
		let (location, dt) = parse_info(
			&row[1],
			&la_config.location,
			&la_config.language,
			&chinese_regex,
		)?;
		let item = TClippingItem {
			content: row[3].clone(),
			title: title,
			pageAt: location,
			createdAt: dt,
		};
		result_list.push(item.clone());
	}

	Ok(result_list)
}

fn parse_title(line: &String) -> String {
	let stop_worlds: Vec<&str> = vec!["(", "（"];
	let mut title = line.clone();
	for w in stop_worlds {
		title = title.split(w).collect();
	}

	title.trim_end_matches(")").to_string()
}

fn parse_info(
	line: &String,
	location_regex: &Regex,
	la: &KindleClippingLanguage,
	chinese_regex: &Regex,
) -> Result<(String, chrono::NaiveDateTime), Box<dyn Error>> {
	let ls: Vec<&str> = line.split("|").collect();
	let location_section = ls[0];

	let matched = location_regex.captures(location_section);
	if matched.is_none() {
		return Err("location not found".into());
	}
	let page_at = matched.unwrap().get(0).unwrap().as_str();
	let dt: chrono::NaiveDateTime;

	let date_section = ls[ls.len() - 1]
		.replace("Added on ", "")
		.replace("添加于 ", "");

	match la {
		KindleClippingLanguage::Zh => {
			let d = chinese_regex.replace_all(&date_section, "-");
			let f = Regex::new(r"-{2,10}").unwrap().replace_all(&d, "");
			// "2006-1-2 3:4:5"
			println!("result str {:?}", f);
			let parsed_dt = chrono::NaiveDateTime::parse_from_str(&f.trim(), "%Y-%m-%e %k:%M:%S")?;
			dt = parsed_dt;
		}
		KindleClippingLanguage::En => {
			let parsed_dt = chrono::NaiveDateTime::parse_from_str(&date_section.trim(), "%A, %B %e, %Y %l:%M:%S %p")?;
			dt = parsed_dt;
		}
	}

	Ok((page_at.to_string(), dt))
}

// const KindleDateTimeLayout {
// 	ENLayout = "Monday, January 2, 2006 3:4:5 PM"
// 	KindleDateTimeZHLayout = "2006-1-2 3:4:5"
// }
