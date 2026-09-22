---
name: list-of-maps
description: A fixture whose frontmatter carries a sequence of mappings.
tools:
  - name: Read
    scope: repo
  - name: Bash
    scope: quarantine
    args:
      - --no-network
      - --read-only
nested-sequence:
  - - a
    - b
  - - c
metadata:
  version: "2.0.0"
---

# list-of-maps

A fixture whose frontmatter carries block sequences whose items are
mappings, including a mapping that itself holds a sequence.
