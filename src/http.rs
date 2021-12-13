use reqwest::Error;
use futures::Future;
use std::collections::HashMap;
use crate::parser::TClippingItem;
use futures;
use reqwest;

pub async fn sync_to_server(
	endpoint: &String,
	result: &Vec<TClippingItem>,
) -> Result<bool, Box<dyn std::error::Error>> {
	// TODO: update this method
	let requestUrl = {
		if endpoint == "http" {
			"http://xxx"
		} else {
			endpoint
		}
	};

	// split chunks
	let mut cursor = 0;

	let client = reqwest::Client::builder().build()?;

	// let reqs: Vec<Future<Output = Result<reqwest::Response, Error>>> = result
	// .chunks(20)
	// .map(|x| {
	// 	let mut params: HashMap<&str, &str> = HashMap::new();
	// 	client.post(endpoint).json(&params).send()
	// })
	// .collect();

	// futures::future::join_all(reqs).await?;

	Ok(true)
}
