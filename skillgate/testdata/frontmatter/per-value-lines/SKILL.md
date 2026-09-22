---
name: per-value-lines
description:
  A plain scalar that begins on the line after its key
  and continues over several lines, folding each break
  into a single space.

  A blank line inside it folds to one newline.
compatibility:
  bash 3.2+
inline-continuation: starts on the key line
  and continues below it
metadata:
  version: "3.1.0"
---

# per-value-lines

A fixture whose frontmatter carries multi-line plain scalars, both
beginning on the key's own line and beginning on the line below it.
