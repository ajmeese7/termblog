use std::cell::RefCell;
use std::io;
use std::rc::Rc;

use ratzilla::event::KeyCode;
use ratzilla::ratatui::layout::{Constraint, Layout};
use ratzilla::ratatui::style::{Modifier, Style};
use ratzilla::ratatui::text::{Line, Span};
use ratzilla::ratatui::widgets::Paragraph;
use ratzilla::ratatui::Terminal;
use ratzilla::{DomBackend, WebRenderer};
use wasm_bindgen_futures::spawn_local;

use crate::api;
use crate::theme::{self, Theme};
use crate::types::{PostDetail, PostSummary};
use crate::views;

#[derive(Debug, Clone, PartialEq)]
pub enum View {
    List,
    Reader,
    Search,
    ThemeSelector,
    Help,
}

pub struct AppState {
    pub view: View,
    pub prev_view: View,

    // List state
    pub posts: Vec<PostSummary>,
    pub cursor: usize,
    pub page: usize,
    pub total_posts: usize,
    pub total_pages: usize,
    pub per_page: usize,

    // Reader state
    pub current_post: Option<PostDetail>,
    pub scroll_offset: usize,

    // Search state
    pub search_query: String,
    pub search_results: Option<Vec<PostSummary>>,
    pub search_cursor: usize,
    pub search_focused: bool,
    pub search_loading: bool,

    // Theme state
    pub theme_index: usize,
    pub theme_cursor: usize,

    // Navigation
    // prev_view: where the reader's back button goes (List or Search)
    // overlay_return: where overlay views (search/theme/help) return on ESC
    pub overlay_return: View,

    // Saved reader state — preserved when search is opened from reader,
    // restored when search ESC returns to reader
    pub saved_post: Option<PostDetail>,
    pub saved_scroll_offset: usize,

    // General
    pub loading: bool,
    pub error: Option<String>,
    pub tick: u64,
    pub ascii_header: String,
    pub blog_title: String,

    // Pending async operations
    pub pending_slug: Option<String>,
}

enum AsyncAction {
    LoadPost(String),
    FetchPosts { page: usize, per_page: usize },
    Search(String),
}

impl AppState {
    pub fn new() -> Self {
        let saved = theme::load_saved_theme().unwrap_or_default();
        let themes = theme::default_themes();
        let idx = themes.iter().position(|t| t.key == saved).unwrap_or(0);

        Self {
            view: View::List,
            prev_view: View::List,
            posts: Vec::new(),
            cursor: 0,
            page: 1,
            total_posts: 0,
            total_pages: 0,
            per_page: 10,
            current_post: None,
            scroll_offset: 0,
            search_query: String::new(),
            search_results: None,
            search_cursor: 0,
            search_focused: true,
            search_loading: false,
            theme_index: idx,
            theme_cursor: idx,
            overlay_return: View::List,
            saved_post: None,
            saved_scroll_offset: 0,
            loading: true,
            error: None,
            tick: 0,
            ascii_header: String::new(),
            blog_title: String::new(),
            pending_slug: None,
        }
    }

    pub fn current_theme(&self) -> Theme {
        // When browsing themes, preview the cursor theme for the whole UI
        let idx = if self.view == View::ThemeSelector {
            self.theme_cursor
        } else {
            self.theme_index
        };
        let themes = theme::default_themes();
        themes
            .into_iter()
            .nth(idx)
            .unwrap_or_else(|| theme::get_theme("pipboy"))
    }
}

pub struct App;

impl App {
    pub fn run() -> io::Result<()> {
        let state = Rc::new(RefCell::new(AppState::new()));

        // Apply initial theme background
        {
            let s = state.borrow();
            let t = s.current_theme();
            theme::set_page_background(&t.colors.background_hex);
        }

        // Load initial data
        {
            let state_clone = state.clone();
            spawn_local(async move {
                // Load config first
                if let Ok(config) = api::get_config().await {
                    let mut s = state_clone.borrow_mut();
                    s.blog_title = config.title;
                    s.ascii_header = config.ascii_header;

                    // Apply default theme from config if no saved preference
                    if theme::load_saved_theme().is_none() && !config.default_theme.is_empty() {
                        let themes = theme::default_themes();
                        if let Some(idx) = themes.iter().position(|t| t.key == config.default_theme)
                        {
                            s.theme_index = idx;
                            s.theme_cursor = idx;
                            let t = themes.into_iter().nth(s.theme_index).unwrap();
                            theme::set_page_background(&t.colors.background_hex);
                        }
                    }
                }

                // Load posts
                match api::get_posts(1, 10).await {
                    Ok(post_list) => {
                        let mut s = state_clone.borrow_mut();
                        s.posts = post_list.posts;
                        s.total_posts = post_list.total;
                        s.total_pages = post_list.total_pages;
                        s.page = post_list.page;
                        s.loading = false;
                    }
                    Err(e) => {
                        let mut s = state_clone.borrow_mut();
                        s.error = Some(format!("Failed to load posts: {}", e));
                        s.loading = false;
                    }
                }
            });
        }

        let backend = DomBackend::new()?;
        let terminal = Terminal::new(backend)?;

        // Keyboard handler
        terminal.on_key_event({
            let state = state.clone();
            move |key_event| {
                // Collect async actions to perform after releasing the borrow
                let action = {
                    let mut s = state.borrow_mut();
                    s.tick += 1;

                    // Capture the slug before handle_key mutates state
                    let prev_view = s.view.clone();
                    handle_key(&mut s, key_event.code, key_event.ctrl, key_event.shift);

                    // Determine what async work is needed
                    if s.view == View::Reader && s.current_post.is_none() && prev_view != View::Reader {
                        // Need to load a post - find the slug
                        let slug = s.pending_slug.take();
                        slug.map(AsyncAction::LoadPost)
                    } else if s.loading && s.view == View::List {
                        Some(AsyncAction::FetchPosts { page: s.page, per_page: s.per_page })
                    } else if s.search_loading && !s.search_query.is_empty() {
                        Some(AsyncAction::Search(s.search_query.clone()))
                    } else {
                        None
                    }
                };

                // Perform async work outside the borrow
                if let Some(action) = action {
                    match action {
                        AsyncAction::LoadPost(slug) => {
                            let state = state.clone();
                            spawn_local(async move {
                                // Record view
                                let _ = api::record_view(&slug).await;

                                match api::get_post(&slug).await {
                                    Ok(post) => {
                                        let mut s = state.borrow_mut();
                                        s.current_post = Some(post);
                                    }
                                    Err(e) => {
                                        let mut s = state.borrow_mut();
                                        s.error = Some(format!("Failed to load post: {}", e));
                                        s.view = View::List;
                                    }
                                }
                            });
                        }
                        AsyncAction::FetchPosts { page, per_page } => {
                            let state = state.clone();
                            spawn_local(async move {
                                match api::get_posts(page, per_page).await {
                                    Ok(post_list) => {
                                        let mut s = state.borrow_mut();
                                        s.posts = post_list.posts;
                                        s.total_posts = post_list.total;
                                        s.total_pages = post_list.total_pages;
                                        s.page = post_list.page;
                                        s.loading = false;
                                    }
                                    Err(e) => {
                                        let mut s = state.borrow_mut();
                                        s.error = Some(format!("Failed to load posts: {}", e));
                                        s.loading = false;
                                    }
                                }
                            });
                        }
                        AsyncAction::Search(query) => {
                            let state = state.clone();
                            spawn_local(async move {
                                match api::search(&query, 20).await {
                                    Ok(result) => {
                                        let mut s = state.borrow_mut();
                                        let has_results = !result.results.is_empty();
                                        s.search_results = Some(result.results);
                                        s.search_cursor = 0;
                                        s.search_loading = false;
                                        // Auto-focus results so user can immediately navigate
                                        if has_results {
                                            s.search_focused = false;
                                        }
                                    }
                                    Err(_) => {
                                        let mut s = state.borrow_mut();
                                        s.search_results = Some(Vec::new());
                                        s.search_loading = false;
                                    }
                                }
                            });
                        }
                    }
                }
            }
        });

        // Render loop
        terminal.draw_web({
            let state = state.clone();
            move |f| {
                let s = state.borrow();
                let area = f.area();
                let colors = &s.current_theme().colors;
                let bg = Style::default().bg(colors.background);

                // Fill entire area with background
                f.render_widget(Paragraph::new("").style(bg), area);

                // Calculate header height based on ASCII header
                let header_lines = if s.ascii_header.is_empty() {
                    1 // just the title
                } else {
                    s.ascii_header.lines().count() as u16
                };

                // Layout: header + content + footer
                let chunks = Layout::vertical([
                    Constraint::Length(header_lines),
                    Constraint::Min(0),
                    Constraint::Length(1),
                ])
                .split(area);

                // Render header
                render_header(f, chunks[0], &s);

                // Render view content
                match s.view {
                    View::List => views::list::render(f, chunks[1], &s),
                    View::Reader => views::reader::render(f, chunks[1], &s),
                    View::Search => views::search::render(f, chunks[1], &s),
                    View::ThemeSelector => views::theme_selector::render(f, chunks[1], &s),
                    View::Help => views::help::render(f, chunks[1], &s),
                }

                // Render footer with keybinding hints
                render_footer(f, chunks[2], &s);
            }
        });

        Ok(())
    }
}

fn handle_key(state: &mut AppState, code: KeyCode, ctrl: bool, shift: bool) {
    match state.view {
        View::List => handle_list_key(state, code, ctrl, shift),
        View::Reader => handle_reader_key(state, code, ctrl),
        View::Search => handle_search_key(state, code, ctrl),
        View::ThemeSelector => handle_theme_key(state, code),
        View::Help => handle_help_key(state, code),
    }
}

fn handle_list_key(state: &mut AppState, code: KeyCode, ctrl: bool, shift: bool) {
    match code {
        KeyCode::Char('j') | KeyCode::Down if !ctrl => {
            if state.cursor < state.posts.len().saturating_sub(1) {
                state.cursor += 1;
            }
        }
        KeyCode::Char('k') | KeyCode::Up if !ctrl => {
            state.cursor = state.cursor.saturating_sub(1);
        }
        KeyCode::Char('d') if ctrl => {
            // Half page down
            let half = 5;
            state.cursor = (state.cursor + half).min(state.posts.len().saturating_sub(1));
        }
        KeyCode::Char('u') if ctrl => {
            // Half page up
            let half = 5;
            state.cursor = state.cursor.saturating_sub(half);
        }
        KeyCode::Char('f') if ctrl => {
            // Full page down
            let page = 10;
            state.cursor = (state.cursor + page).min(state.posts.len().saturating_sub(1));
        }
        KeyCode::Char('b') if ctrl => {
            // Full page up
            let page = 10;
            state.cursor = state.cursor.saturating_sub(page);
        }
        KeyCode::PageDown => {
            let page = 10;
            state.cursor = (state.cursor + page).min(state.posts.len().saturating_sub(1));
        }
        KeyCode::PageUp => {
            let page = 10;
            state.cursor = state.cursor.saturating_sub(page);
        }
        KeyCode::Char('g') if !shift => {
            state.cursor = 0;
        }
        KeyCode::Char('G') => {
            state.cursor = state.posts.len().saturating_sub(1);
        }
        KeyCode::Home => {
            state.cursor = 0;
        }
        KeyCode::End => {
            state.cursor = state.posts.len().saturating_sub(1);
        }
        KeyCode::Enter | KeyCode::Char('l') if !ctrl => {
            if let Some(post) = state.posts.get(state.cursor) {
                let slug = post.slug.clone();
                state.prev_view = View::List;
                state.view = View::Reader;
                state.scroll_offset = 0;
                state.current_post = None; // will be loaded async

                // We can't spawn_local here since we have &mut state
                // Set a flag for the outer handler
                load_post_async(slug, state);
            }
        }
        KeyCode::Char('n') if !ctrl => {
            // Next page
            if state.page < state.total_pages && !state.loading {
                state.page += 1;
                state.cursor = 0;
                state.loading = true;
            }
        }
        KeyCode::Char('p') if !ctrl => {
            // Previous page
            if state.page > 1 && !state.loading {
                state.page -= 1;
                state.cursor = 0;
                state.loading = true;
            }
        }
        KeyCode::Char('/') => {
            state.overlay_return = View::List;
            state.view = View::Search;
            state.search_query.clear();
            state.search_results = None;
            state.search_cursor = 0;
            state.search_focused = true;
        }
        KeyCode::Char('t') if !ctrl => {
            state.overlay_return = View::List;
            state.view = View::ThemeSelector;
            state.theme_cursor = state.theme_index;
        }
        KeyCode::Char('?') => {
            state.overlay_return = View::List;
            state.view = View::Help;
        }
        _ => {}
    }
}

fn handle_reader_key(state: &mut AppState, code: KeyCode, ctrl: bool) {
    match code {
        KeyCode::Esc | KeyCode::Char('h') | KeyCode::Backspace => {
            state.view = state.prev_view.clone();
            if state.view == View::Search {
                // After search→reader cycle, the next reader ESC must not
                // loop back to search. Set prev_view to List so:
                // Reader ESC → Search, Search ESC → Reader, Reader ESC → List
                state.prev_view = View::List;
            }
        }
        KeyCode::Char('j') | KeyCode::Down if !ctrl => {
            state.scroll_offset += 1;
        }
        KeyCode::Char('k') | KeyCode::Up if !ctrl => {
            state.scroll_offset = state.scroll_offset.saturating_sub(1);
        }
        KeyCode::Char('d') if ctrl => {
            state.scroll_offset += 10;
        }
        KeyCode::Char('u') if ctrl => {
            state.scroll_offset = state.scroll_offset.saturating_sub(10);
        }
        KeyCode::Char('f') if ctrl | matches!(code, KeyCode::PageDown) => {
            state.scroll_offset += 20;
        }
        KeyCode::Char('b') if ctrl | matches!(code, KeyCode::PageUp) => {
            state.scroll_offset = state.scroll_offset.saturating_sub(20);
        }
        KeyCode::PageDown => {
            state.scroll_offset += 20;
        }
        KeyCode::PageUp => {
            state.scroll_offset = state.scroll_offset.saturating_sub(20);
        }
        KeyCode::Char('g') => {
            state.scroll_offset = 0;
        }
        KeyCode::Char('G') | KeyCode::End => {
            state.scroll_offset = usize::MAX; // clamped during render
        }
        KeyCode::Home => {
            state.scroll_offset = 0;
        }
        KeyCode::Char('/') => {
            // Save reader state so it can be restored when search closes
            state.saved_post = state.current_post.clone();
            state.saved_scroll_offset = state.scroll_offset;
            state.overlay_return = View::Reader;
            state.view = View::Search;
            state.search_query.clear();
            state.search_results = None;
            state.search_cursor = 0;
            state.search_focused = true;
        }
        KeyCode::Char('t') if !ctrl => {
            state.overlay_return = View::Reader;
            state.view = View::ThemeSelector;
            state.theme_cursor = state.theme_index;
        }
        KeyCode::Char('?') => {
            state.overlay_return = View::Reader;
            state.view = View::Help;
        }
        _ => {}
    }
}

fn handle_search_key(state: &mut AppState, code: KeyCode, ctrl: bool) {
    match code {
        KeyCode::Esc => {
            state.view = state.overlay_return.clone();
            // Restore reader state from before search was opened
            if state.view == View::Reader {
                if let Some(post) = state.saved_post.take() {
                    state.current_post = Some(post);
                    state.scroll_offset = state.saved_scroll_offset;
                }
            }
        }
        KeyCode::Tab => {
            if state.search_focused && state.search_results.as_ref().map_or(false, |r| !r.is_empty()) {
                state.search_focused = false;
            } else {
                state.search_focused = true;
            }
        }
        KeyCode::Enter => {
            if state.search_focused {
                // Trigger search (skip if one is already in-flight)
                if !state.search_query.is_empty() && !state.search_loading {
                    state.search_loading = true;
                }
            } else {
                // Select result — reader's back goes to search
                if let Some(ref results) = state.search_results {
                    if let Some(post) = results.get(state.search_cursor) {
                        let slug = post.slug.clone();
                        state.prev_view = View::Search;
                        state.view = View::Reader;
                        state.scroll_offset = 0;
                        state.current_post = None;
                        load_post_async(slug, state);
                    }
                }
            }
        }
        KeyCode::Char('c') if ctrl => {
            state.search_query.clear();
            state.search_results = None;
        }
        KeyCode::Backspace => {
            if state.search_focused {
                state.search_query.pop();
            }
        }
        KeyCode::Char(c) if state.search_focused => {
            state.search_query.push(c);
        }
        KeyCode::Char('j') | KeyCode::Down if !state.search_focused => {
            if let Some(ref results) = state.search_results {
                let max_visible = 10; // must match views::search::render max_show
                let max_cursor = results.len().min(max_visible).saturating_sub(1);
                if state.search_cursor < max_cursor {
                    state.search_cursor += 1;
                }
            }
        }
        KeyCode::Char('k') | KeyCode::Up if !state.search_focused => {
            state.search_cursor = state.search_cursor.saturating_sub(1);
        }
        _ => {}
    }
}

fn handle_theme_key(state: &mut AppState, code: KeyCode) {
    let theme_count = theme::default_themes().len();

    match code {
        KeyCode::Char('j') | KeyCode::Down => {
            if state.theme_cursor < theme_count - 1 {
                state.theme_cursor += 1;
                // Update page background for live preview
                let t = state.current_theme();
                theme::set_page_background(&t.colors.background_hex);
            }
        }
        KeyCode::Char('k') | KeyCode::Up => {
            if state.theme_cursor > 0 {
                state.theme_cursor -= 1;
                // Update page background for live preview
                let t = state.current_theme();
                theme::set_page_background(&t.colors.background_hex);
            }
        }
        KeyCode::Enter => {
            state.theme_index = state.theme_cursor;
            let t = state.current_theme();
            theme::set_page_background(&t.colors.background_hex);
            theme::save_theme(&t.key);
            state.view = state.overlay_return.clone();
        }
        KeyCode::Esc | KeyCode::Char('h') | KeyCode::Backspace => {
            state.theme_cursor = state.theme_index; // revert
            state.view = state.overlay_return.clone();
            // Restore original theme's background
            let t = state.current_theme();
            theme::set_page_background(&t.colors.background_hex);
        }
        _ => {}
    }
}

fn handle_help_key(state: &mut AppState, code: KeyCode) {
    match code {
        KeyCode::Esc | KeyCode::Char('?') | KeyCode::Char('h') | KeyCode::Backspace => {
            state.view = state.overlay_return.clone();
        }
        _ => {}
    }
}

/// Sets the pending slug so the async handler knows to fetch this post
fn load_post_async(slug: String, state: &mut AppState) {
    state.pending_slug = Some(slug);
}

use ratzilla::ratatui::layout::Rect;
use ratzilla::ratatui::Frame;

fn render_header(f: &mut Frame, area: Rect, state: &AppState) {
    let colors = &state.current_theme().colors;
    let bg = Style::default().bg(colors.background);

    if !state.ascii_header.is_empty() {
        // Render ASCII art header
        let lines: Vec<Line> = state
            .ascii_header
            .lines()
            .map(|l| {
                Line::from(Span::styled(
                    format!("  {}", l),
                    Style::default()
                        .fg(colors.primary)
                        .bg(colors.background)
                        .add_modifier(Modifier::BOLD),
                ))
            })
            .collect();
        f.render_widget(Paragraph::new(lines).style(bg), area);
    } else if !state.blog_title.is_empty() {
        // Render blog title
        let title = Paragraph::new(Line::from(Span::styled(
            format!("  {}", state.blog_title),
            Style::default()
                .fg(colors.primary)
                .bg(colors.background)
                .add_modifier(Modifier::BOLD),
        )))
        .style(bg);
        f.render_widget(title, area);
    } else {
        f.render_widget(Paragraph::new("").style(bg), area);
    }
}

fn render_footer(f: &mut Frame, area: Rect, state: &AppState) {
    let colors = &state.current_theme().colors;
    let bg = Style::default().bg(colors.background);

    let sep = Span::styled(
        "  \u{2502}  ",
        Style::default().fg(colors.border).bg(colors.background),
    );

    let hints: Vec<Vec<Span>> = match state.view {
        View::List => {
            let mut h = vec![
                hint("?", "help", colors),
                hint("/", "search", colors),
                hint("t", "theme", colors),
            ];
            if state.total_pages > 1 {
                h.push(hint("n/p", "page", colors));
            }
            h
        }
        View::Reader => vec![
            hint("esc", "back", colors),
            hint("?", "help", colors),
            hint("/", "search", colors),
            hint("t", "theme", colors),
        ],
        View::Search => vec![
            hint("esc", "cancel", colors),
        ],
        View::ThemeSelector => vec![
            hint("\u{2191}/\u{2193}", "navigate", colors),
            hint("enter", "select", colors),
            hint("esc", "cancel", colors),
        ],
        View::Help => vec![
            hint("esc", "close", colors),
            hint("t", "theme", colors),
        ],
    };

    let mut spans = vec![Span::styled(
        "  ",
        Style::default().bg(colors.background),
    )];
    for (i, h) in hints.into_iter().enumerate() {
        if i > 0 {
            spans.push(sep.clone());
        }
        spans.extend(h);
    }

    f.render_widget(Paragraph::new(Line::from(spans)).style(bg), area);
}

fn hint<'a>(key: &str, desc: &str, colors: &crate::theme::Colors) -> Vec<Span<'a>> {
    vec![
        Span::styled(
            key.to_string(),
            Style::default()
                .fg(colors.accent)
                .bg(colors.background)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            format!(" {}", desc),
            Style::default().fg(colors.muted).bg(colors.background),
        ),
    ]
}

#[cfg(test)]
mod tests {
    use super::*;

    // Tests verify state logic only (no DOM calls), so they run on native targets.
    // handler functions call web_sys DOM APIs, so we test state transitions directly.

    fn test_state() -> AppState {
        AppState {
            view: View::List,
            prev_view: View::List,
            posts: Vec::new(),
            cursor: 0,
            page: 1,
            total_posts: 0,
            total_pages: 0,
            per_page: 10,
            current_post: None,
            scroll_offset: 0,
            search_query: String::new(),
            search_results: None,
            search_cursor: 0,
            search_focused: true,
            search_loading: false,
            theme_index: 0,
            theme_cursor: 0,
            overlay_return: View::List,
            saved_post: None,
            saved_scroll_offset: 0,
            loading: false,
            error: None,
            tick: 0,
            ascii_header: String::new(),
            blog_title: String::new(),
            pending_slug: None,
        }
    }

    #[test]
    fn theme_preview_uses_cursor_in_selector_view() {
        let mut state = test_state();
        state.view = View::ThemeSelector;
        state.theme_index = 0; // confirmed = pipboy
        state.theme_cursor = 1; // browsing = dracula

        let t = state.current_theme();
        assert_eq!(t.key, "dracula", "ThemeSelector view should preview cursor theme");
    }

    #[test]
    fn theme_uses_confirmed_in_list_view() {
        let mut state = test_state();
        state.view = View::List;
        state.theme_index = 0; // confirmed = pipboy
        state.theme_cursor = 3; // stale cursor from earlier browsing

        let t = state.current_theme();
        assert_eq!(t.key, "pipboy", "List view should use confirmed theme_index");
    }

    #[test]
    fn theme_uses_confirmed_in_reader_view() {
        let mut state = test_state();
        state.view = View::Reader;
        state.theme_index = 2; // confirmed = nord
        state.theme_cursor = 5; // stale

        let t = state.current_theme();
        assert_eq!(t.key, "nord", "Reader view should use confirmed theme_index");
    }

    #[test]
    fn theme_cancel_reverts_cursor() {
        let mut state = test_state();
        state.view = View::ThemeSelector;
        state.theme_index = 0; // confirmed = pipboy
        state.theme_cursor = 2; // browsing = nord

        // Simulate what Esc does (without DOM calls)
        state.theme_cursor = state.theme_index;
        state.view = View::List;

        assert_eq!(state.theme_cursor, 0, "Cancel should revert cursor to confirmed index");
        assert_eq!(state.current_theme().key, "pipboy", "Theme should revert on cancel");
    }

    #[test]
    fn theme_confirm_updates_index() {
        let mut state = test_state();
        state.view = View::ThemeSelector;
        state.theme_index = 0;
        state.theme_cursor = 2; // selecting nord

        // Simulate what Enter does (without DOM calls)
        state.theme_index = state.theme_cursor;
        state.view = View::List;

        assert_eq!(state.theme_index, 2, "Enter should confirm cursor as new index");
        assert_eq!(state.current_theme().key, "nord", "Confirmed theme should be nord");
    }

    #[test]
    fn theme_cursor_movement_bounds() {
        let mut state = test_state();
        state.view = View::ThemeSelector;
        state.theme_cursor = 0;

        let theme_count = theme::default_themes().len();

        // Move to last theme
        state.theme_cursor = theme_count - 1;
        assert_eq!(state.theme_cursor, theme_count - 1);

        // Can't go past last theme
        if state.theme_cursor < theme_count - 1 {
            state.theme_cursor += 1;
        }
        assert_eq!(state.theme_cursor, theme_count - 1, "Cursor shouldn't exceed theme count");

        // Can't go below 0
        state.theme_cursor = 0;
        if state.theme_cursor > 0 {
            state.theme_cursor -= 1;
        }
        assert_eq!(state.theme_cursor, 0, "Cursor shouldn't go below 0");
    }

    // --- Navigation tests ---
    // These simulate the exact key handler logic to catch prevView/overlayReturn bugs.
    // Each test traces the full ESC chain to verify no view gets stuck.

    fn make_search_result() -> crate::types::PostSummary {
        crate::types::PostSummary {
            slug: "test".into(),
            title: "Test".into(),
            description: String::new(),
            author: String::new(),
            published_at: String::new(),
            tags: vec![],
            reading_time: 0,
        }
    }

    fn make_post(slug: &str, title: &str) -> PostDetail {
        PostDetail {
            slug: slug.into(),
            title: title.into(),
            description: String::new(),
            author: String::new(),
            content: "Some content".into(),
            tags: vec![],
            published_at: String::new(),
            reading_time: 0,
        }
    }

    #[test]
    fn nav_reader_esc_returns_to_list() {
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;

        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_search_from_list_esc_returns_to_list() {
        let mut state = test_state();
        handle_list_key(&mut state, KeyCode::Char('/'), false, false);
        assert_eq!(state.view, View::Search);

        handle_search_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_search_from_reader_no_selection_full_chain() {
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;

        // `/` from reader
        handle_reader_key(&mut state, KeyCode::Char('/'), false);
        assert_eq!(state.view, View::Search);
        assert_eq!(state.overlay_return, View::Reader);

        // ESC from search → Reader
        handle_search_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::Reader);

        // ESC from reader → List
        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_search_from_list_select_result_full_chain() {
        let mut state = test_state();
        // List → `/`
        handle_list_key(&mut state, KeyCode::Char('/'), false, false);
        assert_eq!(state.view, View::Search);

        // Select result → Reader
        state.search_results = Some(vec![make_search_result()]);
        state.search_focused = false;
        handle_search_key(&mut state, KeyCode::Enter, false);
        assert_eq!(state.view, View::Reader);
        assert_eq!(state.prev_view, View::Search);

        // ESC from reader → Search
        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::Search);

        // ESC from search → List
        handle_search_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_search_from_reader_select_result_full_chain() {
        // Critical test: Reader(A) → Search → Select(B) → Reader(B) → ESC chain
        // Must verify that Post A is restored, not Post B
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;
        state.current_post = Some(make_post("post-a", "Post A")); // reading Post A
        state.scroll_offset = 42;

        // `/` from reader → Search (saves Post A)
        handle_reader_key(&mut state, KeyCode::Char('/'), false);
        assert_eq!(state.view, View::Search);
        assert_eq!(state.overlay_return, View::Reader);
        assert!(state.saved_post.is_some(), "Post A should be saved");
        assert_eq!(state.saved_post.as_ref().unwrap().slug, "post-a");

        // Select a search result → Reader (now showing Post B)
        state.search_results = Some(vec![make_search_result()]);
        state.search_focused = false;
        handle_search_key(&mut state, KeyCode::Enter, false);
        assert_eq!(state.view, View::Reader);
        assert_eq!(state.prev_view, View::Search);
        // Simulate async post load completing for Post B
        state.current_post = Some(make_post("post-b", "Post B"));
        state.scroll_offset = 10;

        // ESC from reader → Search
        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::Search, "First ESC should go to search");

        // ESC from search → Reader (should restore Post A, not Post B)
        handle_search_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::Reader, "Second ESC should go to reader");
        assert_eq!(
            state.current_post.as_ref().unwrap().slug, "post-a",
            "Must restore original Post A, not searched Post B"
        );
        assert_eq!(state.scroll_offset, 42, "Must restore original scroll position");

        // ESC from reader → List
        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List, "Third ESC must reach list, not loop");
    }

    #[test]
    fn nav_search_from_reader_select_never_loops() {
        // Hammer ESC 10 times and verify we always reach List
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;

        // Open search from reader, select result
        handle_reader_key(&mut state, KeyCode::Char('/'), false);
        state.search_results = Some(vec![make_search_result()]);
        state.search_focused = false;
        handle_search_key(&mut state, KeyCode::Enter, false);
        assert_eq!(state.view, View::Reader);

        // Mash ESC up to 10 times — must reach List
        for i in 0..10 {
            if state.view == View::List {
                return; // success
            }
            match state.view {
                View::Reader => handle_reader_key(&mut state, KeyCode::Esc, false),
                View::Search => handle_search_key(&mut state, KeyCode::Esc, false),
                View::Help => handle_help_key(&mut state, KeyCode::Esc),
                _ => { state.view = View::List; }
            }
            assert!(i < 5, "Should reach List in at most 4 ESCs, stuck on {:?}", state.view);
        }
        assert_eq!(state.view, View::List, "Must eventually reach List");
    }

    #[test]
    fn nav_theme_from_reader_returns_to_reader() {
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;

        handle_reader_key(&mut state, KeyCode::Char('t'), false);
        assert_eq!(state.view, View::ThemeSelector);
        assert_eq!(state.overlay_return, View::Reader);

        // Simulate ESC from theme (can't call handle_theme_key — DOM calls)
        state.theme_cursor = state.theme_index;
        state.view = state.overlay_return.clone();
        assert_eq!(state.view, View::Reader);

        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_help_from_reader_returns_to_reader() {
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;

        handle_reader_key(&mut state, KeyCode::Char('?'), false);
        assert_eq!(state.view, View::Help);
        assert_eq!(state.overlay_return, View::Reader);

        handle_help_key(&mut state, KeyCode::Esc);
        assert_eq!(state.view, View::Reader);

        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List);
    }

    #[test]
    fn nav_reader_search_no_select_restores_post() {
        let mut state = test_state();
        state.prev_view = View::List;
        state.view = View::Reader;
        state.current_post = Some(make_post("original", "Original Post"));
        state.scroll_offset = 15;

        // Search from reader without selecting
        handle_reader_key(&mut state, KeyCode::Char('/'), false);
        assert_eq!(state.view, View::Search);

        // ESC back to reader — original post should be restored
        handle_search_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::Reader);
        assert_eq!(
            state.current_post.as_ref().unwrap().slug, "original",
            "Original post must be restored after search cancel"
        );
        assert_eq!(state.scroll_offset, 15, "Scroll position must be restored");

        handle_reader_key(&mut state, KeyCode::Esc, false);
        assert_eq!(state.view, View::List, "Must not get stuck");
    }

    #[test]
    fn theme_preview_changes_with_cursor() {
        let mut state = test_state();
        state.view = View::ThemeSelector;
        state.theme_index = 0;

        // Browse through themes
        state.theme_cursor = 0;
        assert_eq!(state.current_theme().key, "pipboy");

        state.theme_cursor = 1;
        assert_eq!(state.current_theme().key, "dracula");

        state.theme_cursor = 2;
        assert_eq!(state.current_theme().key, "nord");

        // Confirm and leave selector
        state.theme_index = state.theme_cursor;
        state.view = View::List;
        assert_eq!(state.current_theme().key, "nord", "Confirmed theme persists in list view");
    }
}
