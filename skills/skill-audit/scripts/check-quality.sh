#!/usr/bin/env bash
# Check quality: run skillscore for Anthropic-aligned quality scoring.
# Produces a 7-dimension quality report (identity, conciseness, clarity,
# routing, robustness, safety, portability) as JSON.
# Exit codes: 0=success, 3=execution error (skillscore not installed).
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: check-quality.sh <skill-dir>" >&2
  exit 3
fi

skill_dir="$1"

if ! command -v skillscore &>/dev/null; then
  echo "ERROR: skillscore not found. Install with: npm install -g skillscore" >&2
  exit 3
fi

skillscore "$skill_dir" --json
