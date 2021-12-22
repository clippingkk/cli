extern crate dirs;

use crate::constants::CK_ENDPOINT;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::error::Error;
use std::fs;
use std::path::Path;
use std::path::PathBuf;

#[derive(Serialize, Deserialize, Clone)]
pub struct CKConfig {
    pub http: Option<CKConfigHttp>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct CKConfigHttp {
    pub endpoint: Option<String>,
    pub headers: Option<HashMap<String, String>>,
}

impl CKConfig {
    pub fn update_token(self, new_token: &String) -> Result<CKConfig, Box<dyn Error>> {
        let http_endpoint: String;
        let mut http_headers: HashMap<String, String>;

        if let Some(h) = self.http {
            http_endpoint = h.endpoint.unwrap_or(CK_ENDPOINT.to_string());
            http_headers = h.headers.unwrap_or(HashMap::new());
        } else {
            http_endpoint = CK_ENDPOINT.to_string();
            http_headers = HashMap::new();
        }

        let mut token_val = String::from("X-CLI ");
        token_val.push_str(new_token);

        http_headers.insert(String::from("Authorization"), token_val.clone());

        let new_config = CKConfig {
            http: Some(CKConfigHttp {
                endpoint: Some(http_endpoint),
                headers: Some(http_headers),
            }),
        };

        Ok(new_config)
    }

    pub fn save(self, file_path: &Path) -> Result<CKConfig, Box<dyn Error>> {
        let empty_data = toml::to_string(&self)?;
        fs::write(file_path, empty_data)?;
        Ok(self)
    }
}

fn create_empty_config(file_path: &Path) -> Result<CKConfig, Box<dyn Error>> {
    let empty_config = CKConfig {
        http: Some(CKConfigHttp {
            endpoint: Some(CK_ENDPOINT.to_string()),
            headers: None,
        }),
    };
    empty_config.save(file_path)
}

pub fn ensure_toml_config(config_path: &String) -> Result<CKConfig, Box<dyn Error>> {
    let the_config_path = {
        let mut p = dirs::home_dir().unwrap();
        if config_path.is_empty() {
            p.push(".ck-cli.toml");
        } else {
            if config_path.starts_with("~") {
                p.push(config_path.replace("~/", ""))
            } else {
                p = PathBuf::from(config_path);
            }
        }
        let pp = p.clone();
        pp
    };

    let the_config: CKConfig;

    let config_exist = the_config_path.exists();
    if !config_exist {
        if !config_path.is_empty() {
            return Err("config not found".into());
        }
        // 创建
        the_config = create_empty_config(&the_config_path)?;
    } else {
        let config_str = fs::read_to_string(the_config_path)?;
        the_config = toml::from_str(&config_str)?;
    }

    Ok(the_config)
}
