use ratzilla::ratatui::style::Color;

#[derive(Debug, Clone)]
pub struct Theme {
    pub key: String,
    pub name: String,
    pub description: String,
    pub colors: Colors,
}

#[derive(Debug, Clone)]
pub struct Colors {
    pub primary: Color,
    pub secondary: Color,
    pub background: Color,
    pub text: Color,
    pub muted: Color,
    pub accent: Color,
    pub error: Color,
    pub success: Color,
    pub warning: Color,
    pub border: Color,
    // Raw hex for setting page background via DOM
    pub background_hex: String,
}

fn hex_to_color(hex: &str) -> Color {
    if hex.is_empty() {
        return Color::Reset;
    }
    let hex = hex.trim_start_matches('#');
    if hex.len() != 6 {
        return Color::Reset;
    }
    let r = u8::from_str_radix(&hex[0..2], 16).unwrap_or(0);
    let g = u8::from_str_radix(&hex[2..4], 16).unwrap_or(0);
    let b = u8::from_str_radix(&hex[4..6], 16).unwrap_or(0);
    Color::Rgb(r, g, b)
}

pub fn default_themes() -> Vec<Theme> {
    vec![
        Theme {
            key: "pipboy".into(),
            name: "Pip-Boy".into(),
            description: "Retro green terminal aesthetic inspired by Fallout".into(),
            colors: Colors {
                primary: hex_to_color("#00ff00"),
                secondary: hex_to_color("#00cc00"),
                background: hex_to_color("#0a0a0a"),
                text: hex_to_color("#00ff00"),
                muted: hex_to_color("#006600"),
                accent: hex_to_color("#33ff33"),
                error: hex_to_color("#ff3333"),
                success: hex_to_color("#00ff00"),
                warning: hex_to_color("#ffcc00"),
                border: hex_to_color("#00aa00"),
                background_hex: "#0a0a0a".into(),
            },
        },
        Theme {
            key: "dracula".into(),
            name: "Dracula".into(),
            description: "A dark theme with vibrant colors".into(),
            colors: Colors {
                primary: hex_to_color("#bd93f9"),
                secondary: hex_to_color("#ff79c6"),
                background: hex_to_color("#282a36"),
                text: hex_to_color("#f8f8f2"),
                muted: hex_to_color("#6272a4"),
                accent: hex_to_color("#50fa7b"),
                error: hex_to_color("#ff5555"),
                success: hex_to_color("#50fa7b"),
                warning: hex_to_color("#ffb86c"),
                border: hex_to_color("#44475a"),
                background_hex: "#282a36".into(),
            },
        },
        Theme {
            key: "nord".into(),
            name: "Nord".into(),
            description: "An arctic, north-bluish color palette".into(),
            colors: Colors {
                primary: hex_to_color("#88c0d0"),
                secondary: hex_to_color("#81a1c1"),
                background: hex_to_color("#2e3440"),
                text: hex_to_color("#eceff4"),
                muted: hex_to_color("#4c566a"),
                accent: hex_to_color("#a3be8c"),
                error: hex_to_color("#bf616a"),
                success: hex_to_color("#a3be8c"),
                warning: hex_to_color("#ebcb8b"),
                border: hex_to_color("#3b4252"),
                background_hex: "#2e3440".into(),
            },
        },
        Theme {
            key: "monokai".into(),
            name: "Monokai".into(),
            description: "The classic Monokai color scheme".into(),
            colors: Colors {
                primary: hex_to_color("#f92672"),
                secondary: hex_to_color("#66d9ef"),
                background: hex_to_color("#272822"),
                text: hex_to_color("#f8f8f2"),
                muted: hex_to_color("#75715e"),
                accent: hex_to_color("#a6e22e"),
                error: hex_to_color("#f92672"),
                success: hex_to_color("#a6e22e"),
                warning: hex_to_color("#e6db74"),
                border: hex_to_color("#49483e"),
                background_hex: "#272822".into(),
            },
        },
        Theme {
            key: "monochrome".into(),
            name: "Monochrome".into(),
            description: "Pure black and white minimalist theme".into(),
            colors: Colors {
                primary: hex_to_color("#ffffff"),
                secondary: hex_to_color("#cccccc"),
                background: hex_to_color("#000000"),
                text: hex_to_color("#ffffff"),
                muted: hex_to_color("#666666"),
                accent: hex_to_color("#ffffff"),
                error: hex_to_color("#ff0000"),
                success: hex_to_color("#ffffff"),
                warning: hex_to_color("#ffffff"),
                border: hex_to_color("#444444"),
                background_hex: "#000000".into(),
            },
        },
        Theme {
            key: "amber".into(),
            name: "Amber".into(),
            description: "Classic amber CRT terminal aesthetic".into(),
            colors: Colors {
                primary: hex_to_color("#ffb000"),
                secondary: hex_to_color("#ff8c00"),
                background: hex_to_color("#0d0800"),
                text: hex_to_color("#ffb000"),
                muted: hex_to_color("#805800"),
                accent: hex_to_color("#ffc740"),
                error: hex_to_color("#ff4500"),
                success: hex_to_color("#ffb000"),
                warning: hex_to_color("#ffd700"),
                border: hex_to_color("#996600"),
                background_hex: "#0d0800".into(),
            },
        },
        Theme {
            key: "matrix".into(),
            name: "Matrix".into(),
            description: "Green on black digital rain aesthetic".into(),
            colors: Colors {
                primary: hex_to_color("#00ff41"),
                secondary: hex_to_color("#008f11"),
                background: hex_to_color("#0d0208"),
                text: hex_to_color("#00ff41"),
                muted: hex_to_color("#006b00"),
                accent: hex_to_color("#39ff14"),
                error: hex_to_color("#ff0000"),
                success: hex_to_color("#00ff41"),
                warning: hex_to_color("#00ff41"),
                border: hex_to_color("#006400"),
                background_hex: "#0d0208".into(),
            },
        },
        Theme {
            key: "paper".into(),
            name: "Paper".into(),
            description: "Light theme with thermal printer aesthetic".into(),
            colors: Colors {
                primary: hex_to_color("#1a1a1a"),
                secondary: hex_to_color("#4a4a4a"),
                background: hex_to_color("#f5f5dc"),
                text: hex_to_color("#1a1a1a"),
                muted: hex_to_color("#8b8b7a"),
                accent: hex_to_color("#2e2e2e"),
                error: hex_to_color("#8b0000"),
                success: hex_to_color("#1a1a1a"),
                warning: hex_to_color("#654321"),
                border: hex_to_color("#c0c0a8"),
                background_hex: "#f5f5dc".into(),
            },
        },
        Theme {
            key: "terminal".into(),
            name: "Terminal".into(),
            description: "Uses your terminal's native colors".into(),
            colors: Colors {
                primary: Color::LightBlue,
                secondary: Color::LightCyan,
                background: Color::Reset,
                text: Color::Reset,
                muted: Color::DarkGray,
                accent: Color::LightGreen,
                error: Color::LightRed,
                success: Color::LightGreen,
                warning: Color::LightYellow,
                border: Color::DarkGray,
                background_hex: "#1e1e1e".into(),
            },
        },
    ]
}

pub fn get_theme(key: &str) -> Theme {
    default_themes()
        .into_iter()
        .find(|t| t.key == key)
        .unwrap_or_else(|| {
            default_themes()
                .into_iter()
                .find(|t| t.key == "pipboy")
                .unwrap()
        })
}

/// Update the page background color via the DOM
pub fn set_page_background(hex: &str) {
    if let Some(window) = web_sys::window() {
        if let Some(document) = window.document() {
            if let Some(body) = document.body() {
                let _ = body.style().set_property("background-color", hex);
            }
            if let Some(el) = document.document_element() {
                if let Some(html) = el.dyn_ref::<web_sys::HtmlElement>() {
                    let _ = html.style().set_property("background-color", hex);
                }
            }
        }
    }
}

/// Update the favicon link to point at the server-rendered SVG for the given
/// theme key. No-op when the page wasn't served with favicon support enabled,
/// or when the configured mode is "image" (image-mode favicons are
/// theme-agnostic and pointless to refetch).
pub fn set_favicon_for_theme(key: &str) {
    let Some(window) = web_sys::window() else { return };
    let Some(document) = window.document() else { return };

    // Read the mode hint the server injected into <head>. If it's missing or
    // the empty string, the feature is disabled and we don't touch the link.
    let mode = document
        .query_selector(r#"meta[name="termblog-favicon-mode"]"#)
        .ok()
        .flatten()
        .and_then(|el| el.get_attribute("content"))
        .unwrap_or_default();
    if mode != "letter" && mode != "emoji" {
        return;
    }

    let Some(link) = document
        .query_selector(r#"link[rel="icon"]"#)
        .ok()
        .flatten()
    else {
        return;
    };

    let href = format!("/favicon?theme={}", urlencode(key));
    let _ = link.set_attribute("href", &href);
}

/// Minimal URL-component encoder for theme keys. The keys we ship are all
/// `[a-z]+`, but encoding is cheap insurance against future themes with
/// awkward characters.
fn urlencode(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for b in s.as_bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' => {
                out.push(*b as char);
            }
            _ => out.push_str(&format!("%{:02X}", b)),
        }
    }
    out
}

/// Save theme key to localStorage
pub fn save_theme(key: &str) {
    if let Some(window) = web_sys::window() {
        if let Ok(Some(storage)) = window.local_storage() {
            let _ = storage.set_item("termblog-theme", key);
        }
    }
}

/// Load theme key from localStorage
pub fn load_saved_theme() -> Option<String> {
    let window = web_sys::window()?;
    let storage = window.local_storage().ok()??;
    storage.get_item("termblog-theme").ok()?
}

use wasm_bindgen::JsCast;
