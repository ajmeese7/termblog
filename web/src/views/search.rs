use ratzilla::ratatui::{
    layout::{Constraint, Layout, Rect},
    style::{Modifier, Style},
    text::{Line, Span},
    widgets::Paragraph,
    Frame,
};

use crate::app::AppState;
use crate::theme::Colors;

pub fn render(f: &mut Frame, area: Rect, state: &AppState) {
    let colors = &state.current_theme().colors;
    let bg = Style::default().bg(colors.background);
    f.render_widget(Paragraph::new("").style(bg), area);

    let chunks = Layout::vertical([
        Constraint::Length(1), // input
        Constraint::Length(1), // hint
        Constraint::Length(1), // spacer
        Constraint::Min(0),   // results
    ])
    .split(area);

    // Search input line
    let cursor = if state.search_focused { "\u{2588}" } else { "" };
    let input_line = Line::from(vec![
        Span::styled(
            "  / ",
            Style::default()
                .fg(colors.accent)
                .bg(colors.background)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            state.search_query.clone(),
            Style::default().fg(colors.text).bg(colors.background),
        ),
        Span::styled(
            cursor.to_string(),
            Style::default().fg(colors.accent).bg(colors.background),
        ),
    ]);
    f.render_widget(
        Paragraph::new(input_line).style(bg),
        chunks[0],
    );

    // Hint line
    let hint_text = if state.search_focused {
        "  Enter to search, Esc to cancel"
    } else {
        "  j/k navigate, Enter to open, Tab to edit query, Esc to cancel"
    };
    let hint = Paragraph::new(Line::from(Span::styled(
        hint_text.to_string(),
        Style::default().fg(colors.muted).bg(colors.background),
    )))
    .style(bg);
    f.render_widget(hint, chunks[1]);

    // Results area
    let results_area = chunks[3];

    if state.search_loading {
        let loading = Paragraph::new(Line::from(Span::styled(
            "  Searching...".to_string(),
            Style::default().fg(colors.muted).bg(colors.background),
        )))
        .style(bg);
        f.render_widget(loading, results_area);
        return;
    }

    if let Some(ref results) = state.search_results {
        if results.is_empty() && !state.search_query.is_empty() {
            let no_results = Paragraph::new(Line::from(Span::styled(
                "  No results found.".to_string(),
                Style::default().fg(colors.muted).bg(colors.background),
            )))
            .style(bg);
            f.render_widget(no_results, results_area);
            return;
        }

        let max_show = 10;
        let mut lines = Vec::new();

        for (i, post) in results.iter().enumerate().take(max_show) {
            let is_selected = i == state.search_cursor;
            render_result_line(&mut lines, &post.title, &post.published_at, &post.tags, is_selected, colors);
        }

        if results.len() > max_show {
            lines.push(Line::from(Span::styled(
                format!("  ... and {} more", results.len() - max_show),
                Style::default().fg(colors.muted).bg(colors.background),
            )));
        }

        let content = Paragraph::new(lines).style(bg);
        f.render_widget(content, results_area);
    }
}

fn render_result_line(
    lines: &mut Vec<Line<'static>>,
    title: &str,
    date: &str,
    tags: &[String],
    selected: bool,
    colors: &Colors,
) {
    let indicator = if selected { "\u{25b8} " } else { "  " };
    let title_style = if selected {
        Style::default()
            .fg(colors.primary)
            .bg(colors.background)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(colors.text).bg(colors.background)
    };

    lines.push(Line::from(vec![
        Span::styled(
            format!("  {}", indicator),
            Style::default().fg(colors.accent).bg(colors.background),
        ),
        Span::styled(title.to_string(), title_style),
    ]));

    // Date + tags
    let mut meta = format!("      {}", date);
    if !tags.is_empty() {
        meta.push_str(&format!("  [{}]", tags.join(", ")));
    }
    lines.push(Line::from(Span::styled(
        meta,
        Style::default().fg(colors.muted).bg(colors.background),
    )));
}
