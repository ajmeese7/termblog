use ratzilla::ratatui::{
    layout::Rect,
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

    let mut lines = Vec::new();

    // Navigation section
    section_header(&mut lines, "Navigation", colors);
    help_line(&mut lines, "j/\u{2193}", "Move down", colors);
    help_line(&mut lines, "k/\u{2191}", "Move up", colors);
    help_line(&mut lines, "ctrl+d", "Half page down", colors);
    help_line(&mut lines, "ctrl+u", "Half page up", colors);
    help_line(&mut lines, "ctrl+f/pgdn", "Page down", colors);
    help_line(&mut lines, "ctrl+b/pgup", "Page up", colors);
    help_line(&mut lines, "g/home", "Go to top", colors);
    help_line(&mut lines, "G/end", "Go to bottom", colors);
    lines.push(Line::from(""));

    // Actions section
    section_header(&mut lines, "Actions", colors);
    help_line(&mut lines, "enter/l", "Select/Open post", colors);
    help_line(&mut lines, "esc/h", "Go back", colors);
    help_line(&mut lines, "/", "Search posts", colors);
    help_line(&mut lines, "n", "Next page", colors);
    help_line(&mut lines, "p", "Previous page", colors);
    help_line(&mut lines, "t", "Cycle theme", colors);
    help_line(&mut lines, "?", "Toggle this help", colors);
    lines.push(Line::from(""));

    // Tips section
    section_header(&mut lines, "Tips", colors);
    lines.push(Line::from(Span::styled(
        "  Close the browser tab to exit".to_string(),
        Style::default().fg(colors.text).bg(colors.background),
    )));

    let content = Paragraph::new(lines).style(bg);
    f.render_widget(content, area);
}

fn section_header(lines: &mut Vec<Line<'static>>, title: &str, colors: &Colors) {
    lines.push(Line::from(Span::styled(
        format!("  {}", title),
        Style::default()
            .fg(colors.accent)
            .bg(colors.background)
            .add_modifier(Modifier::BOLD),
    )));
}

fn help_line(lines: &mut Vec<Line<'static>>, key: &str, desc: &str, colors: &Colors) {
    lines.push(Line::from(vec![
        Span::styled(
            format!("  {:16}", key),
            Style::default()
                .fg(colors.primary)
                .bg(colors.background)
                .add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            desc.to_string(),
            Style::default().fg(colors.text).bg(colors.background),
        ),
    ]));
}
