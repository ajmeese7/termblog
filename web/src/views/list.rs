use ratzilla::ratatui::{
    layout::{Constraint, Layout, Rect},
    style::{Modifier, Style},
    text::{Line, Span},
    widgets::Paragraph,
    Frame,
};


use crate::app::AppState;
use crate::theme::Colors;
use crate::types::PostSummary;

pub fn render(f: &mut Frame, area: Rect, state: &AppState) {
    let colors = &state.current_theme().colors;
    let bg = Style::default().bg(colors.background);
    f.render_widget(Paragraph::new("").style(bg), area);

    // Loading state
    if state.loading {
        let spinner = spinner_frame(state.tick);
        let loading = Paragraph::new(Line::from(vec![
            Span::styled(
                format!("  {} ", spinner),
                Style::default().fg(colors.accent).bg(colors.background),
            ),
            Span::styled(
                "Loading posts...",
                Style::default().fg(colors.muted).bg(colors.background),
            ),
        ]))
        .style(bg);
        f.render_widget(loading, area);
        return;
    }

    // Error state
    if let Some(ref err) = state.error {
        let error = Paragraph::new(Line::from(Span::styled(
            format!("  {}", err),
            Style::default().fg(colors.error).bg(colors.background),
        )))
        .style(bg);
        f.render_widget(error, area);
        return;
    }

    // Empty state
    let posts = &state.posts;
    if posts.is_empty() {
        let empty = Paragraph::new(Line::from(Span::styled(
            "  No posts yet.",
            Style::default().fg(colors.muted).bg(colors.background),
        )))
        .style(bg);
        f.render_widget(empty, area);
        return;
    }

    // Layout: content + scroll info
    let chunks = Layout::vertical([
        Constraint::Min(0),   // content
        Constraint::Length(1), // scroll info
    ])
    .split(area);

    // Post list
    let content_area = chunks[0];
    let lines_per_post = 3; // title + meta + spacer
    let visible_count = (content_area.height as usize) / lines_per_post;
    let visible_count = visible_count.max(1);

    // Calculate scroll offset to keep cursor visible.
    // The offset positions the viewport so the cursor is always on screen.
    let offset = if state.cursor >= visible_count {
        state.cursor - visible_count + 1
    } else {
        0
    };

    let mut lines = Vec::new();

    for (i, post) in posts.iter().enumerate().skip(offset).take(visible_count) {
        let is_selected = i == state.cursor;
        render_post_line(&mut lines, post, is_selected, colors);
    }

    let content = Paragraph::new(lines).style(bg);
    f.render_widget(content, content_area);

    // Scroll/pagination info
    let show_scroll = posts.len() > visible_count;
    let show_pages = state.total_pages > 1;
    if show_scroll || show_pages {
        let mut parts = Vec::new();
        parts.push(Span::styled(
            "  ".to_string(),
            Style::default().bg(colors.background),
        ));
        if show_scroll {
            let visible_end = (offset + visible_count).min(posts.len());
            parts.push(Span::styled(
                format!("{}-{} of {}", offset + 1, visible_end, posts.len()),
                Style::default().fg(colors.muted).bg(colors.background),
            ));
        }
        if show_pages {
            if show_scroll {
                parts.push(Span::styled(
                    "  \u{2502}  ".to_string(),
                    Style::default().fg(colors.border).bg(colors.background),
                ));
            }
            parts.push(Span::styled(
                format!("page {} of {}", state.page, state.total_pages),
                Style::default().fg(colors.muted).bg(colors.background),
            ));
            if state.page < state.total_pages {
                parts.push(Span::styled(
                    "  n".to_string(),
                    Style::default().fg(colors.accent).bg(colors.background).add_modifier(Modifier::BOLD),
                ));
                parts.push(Span::styled(
                    " next".to_string(),
                    Style::default().fg(colors.muted).bg(colors.background),
                ));
            }
            if state.page > 1 {
                parts.push(Span::styled(
                    "  p".to_string(),
                    Style::default().fg(colors.accent).bg(colors.background).add_modifier(Modifier::BOLD),
                ));
                parts.push(Span::styled(
                    " prev".to_string(),
                    Style::default().fg(colors.muted).bg(colors.background),
                ));
            }
        }
        let scroll = Paragraph::new(Line::from(parts)).style(bg);
        f.render_widget(scroll, chunks[1]);
    }
}

fn render_post_line(lines: &mut Vec<Line<'static>>, post: &PostSummary, selected: bool, colors: &Colors) {
    // Title line
    let indicator = if selected { "\u{25b8} " } else { "  " };
    let title_style = if selected {
        Style::default()
            .fg(colors.primary)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(colors.text)
    };

    lines.push(Line::from(vec![
        Span::styled(
            format!("  {}", indicator),
            Style::default().fg(colors.accent),
        ),
        Span::styled(post.title.clone(), title_style),
    ]));

    // Metadata line: date + tags
    let mut meta_parts = Vec::new();
    meta_parts.push(Span::styled(
        "      ".to_string(),
        Style::default().fg(colors.muted),
    ));
    if !post.published_at.is_empty() {
        meta_parts.push(Span::styled(
            post.published_at.clone(),
            Style::default().fg(colors.muted),
        ));
    }
    if !post.tags.is_empty() {
        meta_parts.push(Span::styled(
            format!("  [{}]", post.tags.join(", ")),
            Style::default().fg(colors.muted),
        ));
    }

    lines.push(Line::from(meta_parts));

    // Spacer line
    lines.push(Line::from(""));
}

fn spinner_frame(tick: u64) -> char {
    const FRAMES: &[char] = &['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'];
    FRAMES[(tick as usize) % FRAMES.len()]
}
