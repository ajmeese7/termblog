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

    let post = match &state.current_post {
        Some(p) => p,
        None => {
            let loading = Paragraph::new(Line::from(Span::styled(
                "Loading...",
                Style::default().fg(colors.muted),
            )))
            .style(bg);
            f.render_widget(loading, area);
            return;
        }
    };

    // Layout: title(1) + meta(1) + blank(1) + content(min) + scroll(1)
    let chunks = Layout::vertical([
        Constraint::Length(1), // title
        Constraint::Length(1), // meta
        Constraint::Length(1), // blank
        Constraint::Min(0),   // content
        Constraint::Length(1), // scroll indicator
    ])
    .split(area);

    // Title bar
    let title = Paragraph::new(Line::from(Span::styled(
        format!("  {}", post.title),
        Style::default()
            .fg(colors.primary)
            .bg(colors.background)
            .add_modifier(Modifier::BOLD),
    )))
    .style(Style::default().bg(colors.background));
    f.render_widget(title, chunks[0]);

    // Meta bar: date + tags
    let mut meta_spans = vec![Span::styled(
        "  ".to_string(),
        Style::default().fg(colors.muted).bg(colors.background),
    )];
    if !post.published_at.is_empty() {
        meta_spans.push(Span::styled(
            post.published_at.clone(),
            Style::default().fg(colors.muted).bg(colors.background),
        ));
    }
    if !post.tags.is_empty() {
        meta_spans.push(Span::styled(
            format!(" \u{2022} {}", post.tags.join(", ")),
            Style::default().fg(colors.muted).bg(colors.background),
        ));
    }
    if post.reading_time > 0 {
        meta_spans.push(Span::styled(
            format!(" \u{2022} {} min read", post.reading_time),
            Style::default().fg(colors.muted).bg(colors.background),
        ));
    }
    let meta = Paragraph::new(Line::from(meta_spans))
        .style(Style::default().bg(colors.background));
    f.render_widget(meta, chunks[1]);

    // Blank line
    f.render_widget(
        Paragraph::new("").style(Style::default().bg(colors.background)),
        chunks[2],
    );

    // Content area: render markdown as simple styled text
    // Strip leading H1 if it matches the title (already shown in the title bar above)
    let content = strip_leading_h1(&post.content, &post.title);
    let content_height = chunks[3].height as usize;
    let content_lines = render_markdown(&content, colors, chunks[3].width as usize);
    let total_lines = content_lines.len();

    // Clamp scroll offset
    let max_offset = total_lines.saturating_sub(content_height);
    let offset = state.scroll_offset.min(max_offset);

    let visible_lines: Vec<Line> = content_lines
        .into_iter()
        .skip(offset)
        .take(content_height)
        .collect();

    let content = Paragraph::new(visible_lines).style(bg);
    f.render_widget(content, chunks[3]);

    // Scroll indicator
    let scroll_pct = if total_lines <= content_height {
        100
    } else if offset == 0 {
        0
    } else {
        (offset * 100) / max_offset
    };

    let scroll_text = format!("  {}%", scroll_pct);
    let scroll = Paragraph::new(Line::from(Span::styled(
        scroll_text,
        Style::default().fg(colors.muted).bg(colors.background),
    )))
    .style(Style::default().bg(colors.background));
    f.render_widget(scroll, chunks[4]);
}

/// Simple markdown-to-styled-lines renderer
fn render_markdown(content: &str, colors: &Colors, width: usize) -> Vec<Line<'static>> {
    let mut lines = Vec::new();
    let padding = "  ";
    let mut in_code_block = false;

    for raw_line in content.lines() {
        let trimmed = raw_line.trim();

        // Code fence toggle
        if trimmed.starts_with("```") {
            if !in_code_block {
                // Entering code block — add spacer
                lines.push(Line::from(Span::styled(
                    " ".to_string(),
                    Style::default().bg(colors.background),
                )));
            } else {
                // Leaving code block — add spacer
                lines.push(Line::from(Span::styled(
                    " ".to_string(),
                    Style::default().bg(colors.background),
                )));
            }
            in_code_block = !in_code_block;
            continue;
        }

        // Inside code block — render with distinct styling
        if in_code_block {
            lines.push(Line::from(vec![
                Span::styled(
                    format!("{}│ ", padding),
                    Style::default().fg(colors.border).bg(colors.background),
                ),
                Span::styled(
                    raw_line.to_string(),
                    Style::default().fg(colors.accent).bg(colors.background),
                ),
            ]));
            continue;
        }

        if trimmed.is_empty() {
            lines.push(Line::from(Span::styled(
                " ".to_string(),
                Style::default().fg(colors.text).bg(colors.background),
            )));
            continue;
        }

        // Headings
        if let Some(heading) = trimmed.strip_prefix("### ") {
            lines.push(Line::from(Span::styled(
                " ".to_string(),
                Style::default().bg(colors.background),
            )));
            lines.push(Line::from(Span::styled(
                format!("{}{}", padding, heading),
                Style::default()
                    .fg(colors.accent)
                    .bg(colors.background)
                    .add_modifier(Modifier::BOLD),
            )));
            continue;
        }
        if let Some(heading) = trimmed.strip_prefix("## ") {
            lines.push(Line::from(Span::styled(
                " ".to_string(),
                Style::default().bg(colors.background),
            )));
            lines.push(Line::from(Span::styled(
                format!("{}{}", padding, heading),
                Style::default()
                    .fg(colors.primary)
                    .bg(colors.background)
                    .add_modifier(Modifier::BOLD),
            )));
            continue;
        }
        if let Some(heading) = trimmed.strip_prefix("# ") {
            lines.push(Line::from(Span::styled(
                " ".to_string(),
                Style::default().bg(colors.background),
            )));
            lines.push(Line::from(Span::styled(
                format!("{}{}", padding, heading),
                Style::default()
                    .fg(colors.primary)
                    .bg(colors.background)
                    .add_modifier(Modifier::BOLD | Modifier::UNDERLINED),
            )));
            continue;
        }

        // Blockquotes
        if let Some(quote) = trimmed.strip_prefix("> ") {
            lines.push(Line::from(vec![
                Span::styled(
                    format!("{}│ ", padding),
                    Style::default().fg(colors.muted).bg(colors.background),
                ),
                Span::styled(
                    quote.to_string(),
                    Style::default()
                        .fg(colors.muted)
                        .bg(colors.background)
                        .add_modifier(Modifier::ITALIC),
                ),
            ]));
            continue;
        }

        // Horizontal rules
        if trimmed == "---" || trimmed == "***" || trimmed == "___" {
            let rule = "\u{2500}".repeat((width.saturating_sub(4)).min(60));
            lines.push(Line::from(Span::styled(
                format!("{}{}", padding, rule),
                Style::default().fg(colors.border).bg(colors.background),
            )));
            continue;
        }

        // List items
        if trimmed.starts_with("- ") || trimmed.starts_with("* ") {
            let item = &trimmed[2..];
            lines.push(Line::from(vec![
                Span::styled(
                    format!("{}  \u{2022} ", padding),
                    Style::default().fg(colors.accent).bg(colors.background),
                ),
                Span::styled(
                    item.to_string(),
                    Style::default().fg(colors.text).bg(colors.background),
                ),
            ]));
            continue;
        }

        // Numbered list items
        if trimmed.len() > 2 && trimmed.chars().next().map_or(false, |c| c.is_ascii_digit()) {
            if let Some(rest) = trimmed.split_once(". ") {
                lines.push(Line::from(vec![
                    Span::styled(
                        format!("{}  {}. ", padding, rest.0),
                        Style::default().fg(colors.accent).bg(colors.background),
                    ),
                    Span::styled(
                        rest.1.to_string(),
                        Style::default().fg(colors.text).bg(colors.background),
                    ),
                ]));
                continue;
            }
        }

        // Regular paragraph text — word-wrap with padding applied after
        let content_width = width.saturating_sub(padding.len());
        let wrapped = word_wrap(trimmed, content_width);
        for wl in wrapped {
            lines.push(Line::from(Span::styled(
                format!("{}{}", padding, wl),
                Style::default().fg(colors.text).bg(colors.background),
            )));
        }
    }

    lines
}

/// Strip the leading H1 heading from markdown content if present.
/// The title is already shown in the reader's title bar, so displaying
/// the H1 again is redundant.
fn strip_leading_h1(content: &str, _title: &str) -> String {
    let mut lines = content.lines().peekable();
    // Skip leading blank lines
    while let Some(line) = lines.peek() {
        if line.trim().is_empty() {
            lines.next();
        } else {
            break;
        }
    }
    // Check if the first non-blank line is an H1
    if let Some(first) = lines.peek() {
        let trimmed = first.trim();
        if trimmed.starts_with("# ") && !trimmed.starts_with("## ") {
            lines.next(); // skip the H1
            // Also skip the blank line after the H1 if present
            if let Some(next) = lines.peek() {
                if next.trim().is_empty() {
                    lines.next();
                }
            }
        }
    }
    lines.collect::<Vec<_>>().join("\n")
}

fn word_wrap(text: &str, width: usize) -> Vec<String> {
    if width == 0 || text.len() <= width {
        return vec![text.to_string()];
    }

    let mut result = Vec::new();
    let mut current = String::new();

    for word in text.split_whitespace() {
        if current.is_empty() {
            current = word.to_string();
        } else if current.len() + 1 + word.len() <= width {
            current.push(' ');
            current.push_str(word);
        } else {
            result.push(current);
            current = word.to_string();
        }
    }

    if !current.is_empty() {
        result.push(current);
    }

    if result.is_empty() {
        result.push(text.to_string());
    }

    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strip_h1_basic() {
        let result = strip_leading_h1("# My Title\n\nSome content.", "My Title");
        assert_eq!(result, "Some content.");
    }

    #[test]
    fn strip_h1_preserves_h2() {
        let result = strip_leading_h1("## Section\n\nContent.", "Section");
        assert_eq!(result, "## Section\n\nContent.");
    }

    #[test]
    fn strip_h1_with_leading_blanks() {
        let result = strip_leading_h1("\n\n# Title\n\nContent.", "Title");
        assert_eq!(result, "Content.");
    }

    #[test]
    fn strip_h1_no_heading() {
        let result = strip_leading_h1("Just text.\n\nMore text.", "Title");
        assert_eq!(result, "Just text.\n\nMore text.");
    }

    #[test]
    fn strip_h1_only() {
        let result = strip_leading_h1("# Solo Title", "Solo Title");
        assert_eq!(result, "");
    }

    #[test]
    fn word_wrap_short_text_unchanged() {
        let result = word_wrap("Hello world", 80);
        assert_eq!(result, vec!["Hello world"]);
    }

    #[test]
    fn word_wrap_long_text_splits() {
        let result = word_wrap("one two three four", 10);
        assert_eq!(result.len(), 2);
        assert_eq!(result[0], "one two");
        assert_eq!(result[1], "three four");
    }

    #[test]
    fn render_markdown_code_block_styled() {
        use crate::theme;
        let t = theme::get_theme("dracula");
        let md = "text before\n\n```bash\necho hello\necho world\n```\n\ntext after";
        let lines = render_markdown(md, &t.colors, 80);

        // Find lines containing "echo" — they should have the │ border marker
        let code_lines: Vec<_> = lines
            .iter()
            .filter(|l| {
                let text: String = l.spans.iter().map(|s| s.content.to_string()).collect();
                text.contains("echo")
            })
            .collect();

        assert_eq!(code_lines.len(), 2, "Should have 2 code lines with 'echo'");
        // Each code line should have at least 2 spans (border + content)
        for cl in &code_lines {
            assert!(cl.spans.len() >= 2, "Code lines should have border + content spans");
            let border: String = cl.spans[0].content.to_string();
            assert!(border.contains('│'), "Code lines should have │ border");
        }
    }

    #[test]
    fn render_markdown_paragraph_padding_consistent() {
        use crate::theme;
        let t = theme::get_theme("dracula");
        // A long paragraph that will need wrapping at width 40
        let md = "This is a somewhat long paragraph that should definitely trigger word wrapping in the renderer.";
        let lines = render_markdown(md, &t.colors, 40);

        // All lines should start with 2-space padding
        for (i, line) in lines.iter().enumerate() {
            let text: String = line.spans.iter().map(|s| s.content.to_string()).collect();
            assert!(
                text.starts_with("  "),
                "Line {} should start with 2-space padding: {:?}",
                i,
                text
            );
        }
    }
}
