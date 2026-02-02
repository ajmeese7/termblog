---
title: "Markdown Showcase"
description: "A demonstration of markdown rendering in the terminal"
author: "Anonymous"
date: 2026-02-02
tags: ["markdown", "demo"]
draft: false
---

# Markdown Showcase

This post demonstrates how various markdown elements render in the terminal.

## Text Formatting

You can use **bold text**, *italic text*, and `inline code`.

You can also combine them: ***bold and italic***.

## Lists

### Unordered Lists

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

### Ordered Lists

1. First step
2. Second step
3. Third step
   1. Sub-step one
   2. Sub-step two

## Blockquotes

> "The best way to predict the future is to invent it."
> — Alan Kay

Nested quotes:

> First level
>> Second level
>>> Third level

## Code Blocks

### Python

```python
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

for i in range(10):
    print(fibonacci(i))
```

### JavaScript

```javascript
const greet = (name) => {
    return `Hello, ${name}!`;
};

console.log(greet("Terminal"));
```

### Bash

```bash
#!/bin/bash
echo "Welcome to the terminal!"
for i in {1..5}; do
    echo "Count: $i"
done
```

## Tables

| Language | Typing | Use Case |
|----------|--------|----------|
| Go | Static | Systems, CLI |
| Python | Dynamic | Scripts, ML |
| Rust | Static | Performance |
| JavaScript | Dynamic | Web |

## Horizontal Rule

---

## Links and References

Check out [Charm](https://charm.sh) for amazing terminal tools.

## Task Lists

- [x] Completed task
- [x] Another completed task
- [ ] Pending task
- [ ] Future task

## Final Notes

Markdown in the terminal is beautiful with Glamour rendering!
