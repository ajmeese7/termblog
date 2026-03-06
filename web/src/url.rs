/// Update the URL hash without triggering a page reload.
/// Pass "/posts/{slug}" to set the hash, or "/" to clear it.
pub fn set_hash(path: &str) {
    #[cfg(target_arch = "wasm32")]
    {
        if let Some(window) = web_sys::window() {
            let hash = if path == "/" {
                String::new()
            } else {
                format!("#{}", path)
            };
            let _ = window.location().set_hash(&hash);
        }
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        let _ = path;
    }
}

/// Parse the current URL hash and extract a post slug if present.
/// Matches `#/posts/{slug}` pattern.
pub fn get_hash_slug() -> Option<String> {
    #[cfg(target_arch = "wasm32")]
    {
        let window = web_sys::window()?;
        let hash = window.location().hash().ok()?;
        let path = hash.strip_prefix('#')?;
        let slug = path.strip_prefix("/posts/")?;
        let slug = slug.trim_end_matches('/');
        if slug.is_empty() {
            None
        } else {
            Some(slug.to_string())
        }
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        None
    }
}

/// Get the page origin (e.g. "https://termblog.com")
pub fn get_origin() -> Option<String> {
    #[cfg(target_arch = "wasm32")]
    {
        let window = web_sys::window()?;
        window.location().origin().ok()
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        None
    }
}

/// Copy text to the clipboard using the Clipboard API.
pub fn copy_to_clipboard(text: &str) {
    #[cfg(target_arch = "wasm32")]
    {
        let text = text.to_string();
        wasm_bindgen_futures::spawn_local(async move {
            if let Some(window) = web_sys::window() {
                let clipboard = window.navigator().clipboard();
                let promise = clipboard.write_text(&text);
                let _ = wasm_bindgen_futures::JsFuture::from(promise).await;
            }
        });
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        let _ = text;
    }
}
