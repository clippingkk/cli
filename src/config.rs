extern crate dirs;

use crate::constants::CK_ENDPOINT;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::error::Error;
use std::fs;
use std::path::Path;

#[derive(Serialize, Deserialize)]
pub struct CKConfig {
    pub http: Option<CKConfigHttp>,
}

#[derive(Serialize, Deserialize)]
pub struct CKConfigHttp {
    pub endpoint: Option<String>,
    pub headers: Option<HashMap<String, String>>,
}

fn create_empty_config(file_path: &Path) -> Result<CKConfig, Box<dyn Error>> {
    let empty_config = CKConfig {
        http: Some(CKConfigHttp {
            endpoint: Some(CK_ENDPOINT.to_string()),
            headers: None,
        }),
    };
    let empty_data = toml::to_string(&empty_config)?;
    fs::write(file_path, empty_data)?;
    Ok(empty_config)
}

pub fn ensure_toml_config(config_path: &String) -> Result<CKConfig, Box<dyn Error>> {
    let the_config_path = {
        if config_path.is_empty() {
            let mut p = dirs::home_dir().unwrap();
            p.push(".ck-cli.toml");
            let pp = p.clone();
            pp
        } else {
            Path::new(config_path).to_path_buf()
        }
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
