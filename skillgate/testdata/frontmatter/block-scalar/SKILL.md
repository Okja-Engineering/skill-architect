---
name: block-scalar
description: |
  Clip chomping keeps one trailing newline.
  A second content line.
summary: |-
  Strip chomping keeps none.
  Second line.
notes: |+
  Keep chomping keeps them all.

indented: |2
     two-space indicator, so three spaces survive.
metadata:
  version: "0.3.0"
---

# block-scalar

A fixture whose frontmatter carries literal block scalars in all four
chomping forms plus an explicit indentation indicator.
