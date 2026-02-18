use wasm_bindgen::prelude::*;
use wasm_bindgen_futures::JsFuture;
use web_sys::js_sys;
use web_sys::{Request, RequestInit, Response};

use crate::types::{BlogConfig, PostDetail, PostList, SearchResult, Tag};

fn window() -> web_sys::Window {
    web_sys::window().expect("no global window")
}

async fn fetch_json(url: &str) -> Result<String, String> {
    let opts = RequestInit::new();
    opts.set_method("GET");

    let request = Request::new_with_str_and_init(url, &opts)
        .map_err(|e| format!("request error: {:?}", e))?;

    let resp_value = JsFuture::from(window().fetch_with_request(&request))
        .await
        .map_err(|e| format!("fetch error: {:?}", e))?;

    let resp: Response = resp_value
        .dyn_into()
        .map_err(|_| "response is not a Response".to_string())?;

    if !resp.ok() {
        return Err(format!("HTTP {}", resp.status()));
    }

    let text = JsFuture::from(
        resp.text()
            .map_err(|e| format!("text error: {:?}", e))?,
    )
    .await
    .map_err(|e| format!("text promise error: {:?}", e))?;

    text.as_string()
        .ok_or_else(|| "response is not a string".to_string())
}

async fn fetch_post(url: &str) -> Result<String, String> {
    let opts = RequestInit::new();
    opts.set_method("POST");

    let request = Request::new_with_str_and_init(url, &opts)
        .map_err(|e| format!("request error: {:?}", e))?;

    let resp_value = JsFuture::from(window().fetch_with_request(&request))
        .await
        .map_err(|e| format!("fetch error: {:?}", e))?;

    let resp: Response = resp_value
        .dyn_into()
        .map_err(|_| "response is not a Response".to_string())?;

    if !resp.ok() {
        return Err(format!("HTTP {}", resp.status()));
    }

    Ok("ok".to_string())
}

pub async fn get_posts(page: usize, per_page: usize) -> Result<PostList, String> {
    let url = format!("/api/posts?page={}&per_page={}", page, per_page);
    let json = fetch_json(&url).await?;
    serde_json::from_str(&json).map_err(|e| format!("parse error: {}", e))
}

pub async fn get_post(slug: &str) -> Result<PostDetail, String> {
    let url = format!("/api/posts/{}", slug);
    let json = fetch_json(&url).await?;
    serde_json::from_str(&json).map_err(|e| format!("parse error: {}", e))
}

pub async fn search(query: &str, limit: usize) -> Result<SearchResult, String> {
    let encoded = js_sys::encode_uri_component(query);
    let url = format!(
        "/api/search?q={}&limit={}",
        encoded.as_string().unwrap_or_default(),
        limit
    );
    let json = fetch_json(&url).await?;
    serde_json::from_str(&json).map_err(|e| format!("parse error: {}", e))
}

pub async fn get_tags() -> Result<Vec<Tag>, String> {
    let json = fetch_json("/api/tags").await?;
    serde_json::from_str(&json).map_err(|e| format!("parse error: {}", e))
}

pub async fn get_config() -> Result<BlogConfig, String> {
    let json = fetch_json("/api/config").await?;
    serde_json::from_str(&json).map_err(|e| format!("parse error: {}", e))
}

pub async fn record_view(slug: &str) -> Result<(), String> {
    let url = format!("/api/views/{}", slug);
    fetch_post(&url).await?;
    Ok(())
}
