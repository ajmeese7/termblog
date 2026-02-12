---
title: "Theming Your Terminal Publication"
description: "Use built-in themes and custom YAML palettes to give your blog a distinct identity"
author: "TermBlog Team"
date: 2026-02-06
tags: ["themes", "design", "customization"]
draft: false
---

# Theming Your Terminal Publication

A terminal interface can still feel branded and deliberate. TermBlog includes multiple built-in themes and supports custom theme files.

## Built-in options

Out of the box, TermBlog ships with themes like:

- Pip-Boy
- Dracula
- Nord
- Monokai
- Monochrome
- Amber
- Matrix
- Paper

Readers can switch themes in-app with `t`, and preferences persist per SSH key fingerprint.

## Custom theme file

Create a YAML file like `themes/night-ops.yaml`:

```yaml
name: "Night Ops"
description: "High-contrast blue and cyan palette"
colors:
  primary: "#c7d2fe"
  secondary: "#93c5fd"
  background: "#0a0f1e"
  text: "#dbeafe"
  muted: "#64748b"
  accent: "#22d3ee"
  error: "#fb7185"
  success: "#34d399"
  warning: "#facc15"
  border: "#1e293b"
```

Then point your config to it:

```yaml
theme: "themes/night-ops.yaml"
```

## Practical tips

- Optimize for contrast first, style second.
- Test both long-form text and dense list screens.
- Keep accents intentional so selected states are obvious.

A strong theme makes your blog feel less like a demo and more like a product.
