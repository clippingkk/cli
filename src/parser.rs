use chrono;
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::error::Error;
use std::vec::Vec;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TClippingItem {
    pub title: String,
    pub content: String,
    #[serde(rename = "pageAt")]
    pub page_at: String,
    #[serde(rename = "createdAt", with = "datetime_parser_with_rfc3339")]
    pub created_at: chrono::NaiveDateTime,
}

mod datetime_parser_with_rfc3339 {
    use chrono::{DateTime, NaiveDateTime, TimeZone, Utc};
    use serde::{self, Deserialize, Deserializer, Serializer};
    pub fn serialize<S>(date: &NaiveDateTime, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let s = Utc.from_local_datetime(date).unwrap().to_rfc3339();
        serializer.serialize_str(&s)
    }
    pub fn deserialize<'de, D>(deserializer: D) -> Result<NaiveDateTime, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        let d = DateTime::parse_from_rfc3339(&s).unwrap();
        Ok(d.naive_utc())
    }
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

    let r = regex::Regex::new(r"\u{feff}").unwrap();
    for row in grouped {
        let content = row[3].trim();
        if content.is_empty() {
            continue;
        }

        let title = parse_title(&r.replace_all(&row[0], "").trim().to_string());
        let (location, dt) = parse_info(
            &row[1],
            &la_config.location,
            &la_config.language,
            &chinese_regex,
        )?;
        let item = TClippingItem {
            content: content.to_string(),
            title: title,
            page_at: location,
            created_at: dt,
        };
        result_list.push(item.clone());
    }

    Ok(result_list)
}

fn parse_title(line: &String) -> String {
    let stop_worlds: Vec<&str> = vec!["(", "（"];
    let mut title = line.clone();
    for w in stop_worlds {
        let nt: Vec<&str> = title.split(w).collect();
        title = nt.first().unwrap().to_string();
    }

    title.trim_end_matches(")").trim().to_string()
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
    let page_at_str = matched.unwrap().get(0).unwrap().as_str();

    let page_at = if page_at_str.starts_with("#") {
        page_at_str.to_string()
    } else {
        ("#".to_owned() + page_at_str).to_string()
    };

    let dt: chrono::NaiveDateTime;

    let date_section = ls[ls.len() - 1]
        .replace("Added on ", "")
        .replace("添加于 ", "");

    match la {
        KindleClippingLanguage::Zh => {
            let clocks_ext = {
                if date_section.contains("上午") {
                    "AM"
                } else {
                    "PM"
                }
            };
            let d = chinese_regex.replace_all(&date_section, "-");
            let mut f = Regex::new(r"-{2,10}")
                .unwrap()
                .replace_all(&d, "")
                .trim()
                .to_owned();
            f.push_str(" ");
            f.push_str(clocks_ext);
            // "2006-1-2 3:4:5"
            let parsed_dt = chrono::NaiveDateTime::parse_from_str(&f, "%Y-%m-%e %l:%M:%S %p")?;
            dt = parsed_dt;
        }
        KindleClippingLanguage::En => {
            let parsed_dt = chrono::NaiveDateTime::parse_from_str(
                &date_section.trim(),
                "%A, %B %e, %Y %l:%M:%S %p",
            )?;
            dt = parsed_dt;
        }
    }
    Ok((page_at.to_string(), dt))
}
