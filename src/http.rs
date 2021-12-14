use crate::constants::CK_ENDPOINT;
use crate::parser::TClippingItem;
use chrono::{TimeZone, Utc};
use futures;
use futures::{stream, StreamExt};
use reqwest;
use reqwest::Client;
use serde::{Deserialize, Serialize};

const PARALLEL_REQUESTS: usize = 10;
const AUTH_HEADER_KEY: &str = "Authorization";

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

pub async fn sync_to_server(
    endpoint: &String,
    jwt: &String,
    result: &Vec<TClippingItem>,
) -> Result<bool, Box<dyn std::error::Error>> {
    let request_url = {
        if endpoint == "http" {
            CK_ENDPOINT.to_string()
        } else {
            endpoint.clone()
        }
    };

    // TODO: add login method
    let mut credential = String::from("Bearer ");
    credential.push_str(jwt);

    let client = Client::new();

    let chunked: Vec<&[TClippingItem]> = result.chunks(20).collect();

    let bodies = stream::iter(chunked)
        .map(|chunk| {
            let c = client.clone();
            let req_url = request_url.clone();
            let v = &chunk.to_vec();
            let token = credential.clone();
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
                            book_id: "",
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
                    .header(AUTH_HEADER_KEY, token)
                    .json(&payload)
                    .send()
                    .await?;
                resp.text().await
            })
        })
        .buffer_unordered(PARALLEL_REQUESTS);

    bodies
        .for_each(|b| async {
            match b {
                Ok(Ok(b)) => println!("success {:?}", b),
                Ok(Err(e)) => eprintln!("failed {:?}", e),
                Err(e) => eprintln!("tokio error {:?}", e),
            }
        })
        .await;

    println!("done");
    Ok(true)
}
