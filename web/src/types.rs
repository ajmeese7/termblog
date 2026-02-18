use serde::Deserialize;

#[derive(Debug, Clone, Deserialize)]
pub struct PostSummary {
    pub slug: String,
    pub title: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub author: String,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(default)]
    pub published_at: String,
    #[serde(default)]
    pub reading_time: u32,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PostDetail {
    pub slug: String,
    pub title: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub author: String,
    pub content: String,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(default)]
    pub published_at: String,
    #[serde(default)]
    pub reading_time: u32,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PostList {
    pub posts: Vec<PostSummary>,
    pub total: usize,
    pub page: usize,
    pub per_page: usize,
    pub total_pages: usize,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SearchResult {
    pub query: String,
    pub results: Vec<PostSummary>,
    pub total: usize,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Tag {
    pub name: String,
    pub count: usize,
}

#[derive(Debug, Clone, Deserialize)]
pub struct BlogConfig {
    pub title: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub author: String,
    #[serde(default)]
    pub themes: Vec<ThemeConfig>,
    #[serde(default)]
    pub default_theme: String,
    #[serde(default)]
    pub ascii_header: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ThemeConfig {
    pub key: String,
    pub name: String,
    #[serde(default)]
    pub description: String,
    pub colors: ThemeColors,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ThemeColors {
    pub primary: String,
    pub secondary: String,
    pub background: String,
    pub text: String,
    pub muted: String,
    pub accent: String,
    pub error: String,
    pub success: String,
    pub warning: String,
    pub border: String,
}
