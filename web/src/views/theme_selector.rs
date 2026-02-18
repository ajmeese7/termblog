use ratzilla::ratatui::{
    layout::{Constraint, Layout, Rect},
    style::{Modifier, Style},
    text::{Line, Span},
    widgets::Paragraph,
    Frame,
};

use crate::app::AppState;
use crate::theme;

pub fn render(f: &mut Frame, area: Rect, state: &AppState) {
    let colors = &state.current_theme().colors;
    let bg = Style::default().bg(colors.background);
    f.render_widget(Paragraph::new("").style(bg), area);

    let chunks = Layout::vertical([
        Constraint::Length(1), // title
        Constraint::Length(1), // spacer
        Constraint::Min(0),   // theme list
    ])
    .split(area);

    // Title
    let title = Paragraph::new(Line::from(Span::styled(
        "  Select Theme".to_string(),
        Style::default()
            .fg(colors.accent)
            .bg(colors.background)
            .add_modifier(Modifier::BOLD),
    )))
    .style(bg);
    f.render_widget(title, chunks[0]);

    // Theme list
    let themes = theme::default_themes();
    let list_area = chunks[2];
    let mut lines = Vec::new();

    for (i, t) in themes.iter().enumerate() {
        let is_selected = i == state.theme_cursor;
        let is_active = t.key == state.current_theme().key;

        // Theme name line
        let indicator = if is_selected { "> " } else { "  " };
        let check = if is_active { " \u{2713}" } else { "" };

        let name_style = if is_selected {
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
            Span::styled(t.name.clone(), name_style),
            Span::styled(
                check.to_string(),
                Style::default().fg(colors.accent).bg(colors.background),
            ),
        ]));

        // Description (only for highlighted theme)
        if is_selected {
            lines.push(Line::from(Span::styled(
                format!("      {}", t.description),
                Style::default()
                    .fg(colors.muted)
                    .bg(colors.background)
                    .add_modifier(Modifier::ITALIC),
            )));

            // Color preview blocks
            let preview = render_color_preview(t);
            lines.push(Line::from(preview));
        }
    }

    let content = Paragraph::new(lines).style(bg);
    f.render_widget(content, list_area);
}

fn render_color_preview(theme: &theme::Theme) -> Vec<Span<'static>> {
    let labels = ["Pri", "Sec", "Acc", "Txt", "Mut"];
    let preview_colors = [
        theme.colors.primary,
        theme.colors.secondary,
        theme.colors.accent,
        theme.colors.text,
        theme.colors.muted,
    ];

    let mut spans = vec![Span::styled(
        "      ".to_string(),
        Style::default().bg(theme.colors.background),
    )];

    for (label, color) in labels.iter().zip(preview_colors.iter()) {
        spans.push(Span::styled(
            format!(" {} ", label),
            Style::default().fg(theme.colors.background).bg(*color),
        ));
        spans.push(Span::styled(
            " ".to_string(),
            Style::default().bg(theme.colors.background),
        ));
    }

    spans
}
