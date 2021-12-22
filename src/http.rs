use crate::config::CKConfigHttp;
use crate::graphql::{CreateClippingsDataResponse, GraphQLResponse};
use crate::parser::TClippingItem;
use chrono::{TimeZone, Utc};
use futures;
use futures::{stream, StreamExt};
use reqwest;
use reqwest::header::{HeaderMap, HeaderName};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::error::Error;
use std::str::FromStr;

const PARALLEL_REQUESTS: usize = 10;

const CREATE_CLIPPINGS_QUERY: &str =
    "mutation createClippings($payload: [ClippingInput!]!, $visible: Boolean) {
    createClippings(payload: $payload, visible: $visible) {
        id
    }
}
";

#[derive(Debug, Clone, Serialize, Deserialize)]
struct TClippingInput {
    title: String,
    content: String,
    #[serde(rename = "bookID")]
    book_id: &'static str,
    #[serde(rename = "pageAt")]
    page_at: String,
    #[serde(rename = "createdAt")]
    created_at: String,
    source: &'static str,
}

#[derive(Debug, Clone, Serialize)]
struct CreateClippingPayload {
    visible: bool,
    payload: Vec<TClippingInput>,
}

#[derive(Debug, Clone, Serialize)]
struct GraphQLPayload<T> {
    #[serde(rename = "operationName")]
    pub operation_name: &'static str,
    pub query: &'static str,
    pub variables: T,
}

fn get_real_endpoint(
    endpoint_in_arg: &String,
    ck_http_config: &Option<CKConfigHttp>,
) -> Result<String, Box<dyn Error>> {
    if !endpoint_in_arg.starts_with("http") {
        return Err("not allowed".into());
    }

    if endpoint_in_arg != "http" {
        return Ok(endpoint_in_arg.clone());
    }

    if let Some(the_http_config) = ck_http_config {
        if let Some(endpoint) = &the_http_config.endpoint {
            return Ok(endpoint.clone());
        }
    }

    return Err("http endpoint not found".into());
}

pub async fn sync_to_server(
    endpoint: &String,
    ck_http_config: &Option<CKConfigHttp>,
    result: &Vec<TClippingItem>,
) -> Result<bool, Box<dyn Error>> {
    let request_url = get_real_endpoint(endpoint, ck_http_config)?;

    let client = Client::new();
    let mut header_maps = HeaderMap::new();

    if let Some(http) = ck_http_config {
        if let Some(hs) = &http.headers {
            for (k, v) in hs {
                header_maps.insert(HeaderName::from_str(&k).unwrap(), v.parse().unwrap());
            }
        }
    }

    let chunked: Vec<&[TClippingItem]> = result.chunks(20).collect();

    let bodies = stream::iter(chunked)
        .map(|chunk| {
            let c = client.clone();
            let req_url = request_url.clone();
            let v = &chunk.to_vec();
            let headers = header_maps.clone();
            let payload = GraphQLPayload::<CreateClippingPayload> {
                operation_name: "createClippings",
                query: CREATE_CLIPPINGS_QUERY,
                variables: CreateClippingPayload {
                    visible: true,
                    payload: v
                        .iter()
                        .map(|x| TClippingInput {
                            title: x.title.clone(),
                            content: x.content.clone(),
                            book_id: "0",
                            page_at: x.page_at.clone(),
                            created_at: Utc
                                .from_local_datetime(&x.created_at)
                                .unwrap()
                                .to_rfc3339(),
                            source: "kindle",
                        })
                        .collect(),
                },
            };
            tokio::spawn(async move {
                let resp = c
                    .post(req_url)
                    .headers(headers)
                    .json(&payload)
                    .send()
                    .await?;
                let r = resp.error_for_status()?;
                r.text().await
            })
        })
        .buffer_unordered(PARALLEL_REQUESTS);

    bodies
        .for_each(|b| async {
            match b {
                Ok(Ok(b)) => {
                    let res: GraphQLResponse<CreateClippingsDataResponse> =
                        serde_json::from_str(&b).unwrap();
                    if let Some(errs) = res.errors {
                        e_red_ln!(
                            " ❌ request to {:?} got errors: {:?}",
                            request_url,
                            errs[0].message
                        )
                    }
                    if let Some(data) = res.data {
                        green_ln!(" ✅ completed: {:?} rows", data.create_clippings.len())
                    }
                }
                Ok(Err(e)) => {
                    e_red_ln!(" ❌ failed {:?}", e)
                }
                Err(e) => {
                    e_red_ln!(" ❌ tokio error {:?}", e)
                }
            }
        })
        .await;
    Ok(true)
}
