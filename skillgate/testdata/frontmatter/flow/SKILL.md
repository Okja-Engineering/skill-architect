---
name: flow
description: A fixture whose frontmatter carries flow collections.
allowed-tools: [Read, Bash]
matrix: {shell: bash, version: "3.2"}
spanning: [
  Read,
  Bash,
]
deep: {tools: [Read, {name: Bash, scope: quarantine}]}
quoted: "a value with: a colon, and a # hash"
single: 'it''s quoted'
escaped: "tab\there\nand a newline"
metadata:
  version: "4.0.0"
---

# flow

A fixture whose frontmatter carries flow mappings and sequences, including
one that spans lines and one nested inside a sequence, plus quoted scalars.
