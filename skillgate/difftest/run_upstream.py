#!/usr/bin/env python3
"""Upstream-side of the E2 differential harness.

Runs NVIDIA SkillSpector's own analyzers (verbatim — no transcription) over a
corpus and emits normalized JSONL records on stdout:

    {"rule": "...", "file": "corpus-relative/path", "line": N, "evidence": "..."}

Usage:
    run_upstream.py --corpus-root <dir> --bundle-root <dir> [<dir>...]

File rules are invoked per file under every --corpus-root. State rules (BH2,
SSR-1) are invoked per immediate child directory under every --bundle-root.
Requires `skillspector` importable (Python 3.12+ venv).
"""
import argparse
import json
import sys
from pathlib import Path, PurePosixPath

from skillspector.models import AnalyzerFinding, Finding  # noqa: F401
from skillspector.nodes.analyzers import (
    static_patterns_agent_snooping,
    static_patterns_anti_refusal,
    static_patterns_excessive_agency,
    static_patterns_harmful_content,
    static_patterns_memory_poisoning,
    static_patterns_output_handling,
    static_patterns_rogue_agent,
    static_patterns_system_prompt_leakage,
    static_patterns_tool_misuse,
    artifact_integrity,
    bundled_execution_surface,
    mcp_tool_poisoning,
    structured_skill_roles,
    whitespace_padding,
)
from skillspector.structured_skill import extract_structured_skill_context

# Same extension map as static_runner.FILE_TYPES.
FILE_TYPES = {
    ".md": "markdown", ".markdown": "markdown", ".py": "python",
    ".sh": "shell", ".bash": "shell", ".zsh": "shell", ".json": "json",
    ".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".txt": "text",
    ".js": "javascript", ".ts": "typescript", ".rb": "ruby",
}

# rule_id -> module exposing analyze(content, file_path, file_type)
FILE_ANALYZERS = {
    "EA1": static_patterns_excessive_agency,
    "P5": static_patterns_harmful_content,
    "MP2": static_patterns_memory_poisoning,
    "OH1": static_patterns_output_handling,
    "TM1": static_patterns_tool_misuse,
    "RA1": static_patterns_rogue_agent,
    "P6": static_patterns_system_prompt_leakage,
    "AS3": static_patterns_agent_snooping,
    "AR2": static_patterns_anti_refusal,
}

# Upstream runs OH1's AST lane for python files; the Go port re-expresses the
# regex fallback lane only. The runner still records upstream output — the
# differ attributes those rows to the declared cede.


def _emit(out, rule, rel, line, evidence):
    out.append({"rule": rule, "file": rel, "line": int(line or 0),
                "evidence": (evidence or "")[:400]})


def _run_file_rules(content, rel, ftype, out):
    for rule, mod in FILE_ANALYZERS.items():
        try:
            findings = mod.analyze(content=content, file_path=rel, file_type=ftype)
        except Exception as exc:  # honest: record analyzer failure, keep going
            _emit(out, rule, rel, 0, f"ANALYZER_ERROR:{type(exc).__name__}")
            continue
        for f in findings:
            if f.rule_id != rule:
                continue
            _emit(out, rule, rel, f.location.start_line, f.matched_text)


_TP1_LANES = {
    "Data URI found": "datauri",
    "HTML comment found": "html-comment",
    "Markdown comment found": "md-comment",
    "Zero-width character(s)": "zero-width",
    "Base64-encoded blob": "base64",
}


def _run_tp1(content, rel, out):
    try:
        findings = mcp_tool_poisoning._check_tp1(content, "content")
    except Exception as exc:
        _emit(out, "TP1", rel, 0, f"ANALYZER_ERROR:{type(exc).__name__}")
        return
    for f in findings:
        if f.rule_id != "TP1":
            continue
        lane = next((v for k, v in _TP1_LANES.items() if f.message.startswith(k)), "other")
        # Finding has no line; recover offset via the evidence prefix.
        probe = (f.matched_text or "")[:80].rstrip(".")
        pos = content.find(probe) if probe else -1
        line = content.count("\n", 0, pos) + 1 if pos >= 0 else 0
        _emit(out, f"TP1:{lane}", rel, line, f.matched_text)


def _run_p9(content, rel, ftype, out):
    try:
        for run in whitespace_padding.detect_whitespace_padding(content, file_type=ftype):
            _emit(out, "P9", rel, run.start_line,
                  f"{run.kind}@{run.start_offset}:{run.length}")
    except Exception as exc:
        _emit(out, "P9", rel, 0, f"ANALYZER_ERROR:{type(exc).__name__}")


class _StubBudget:
    """Minimal stand-in for _ArtifactIntegrityBudget: no limits, collect emits."""

    def __init__(self):
        self.findings = []

    def check_runtime(self):
        return True

    def emit(self, finding):
        self.findings.append(finding)

    def analyzer_exhausted(self):
        return False


def _run_ae6_lanes(content, rel, out):
    """Emit per-lane AE6 records; the Go port covers the letter-spacing lane."""
    ai = artifact_integrity
    budget = _StubBudget()
    try:
        for span in ai._concealed_instruction_run_spans(content, budget.check_runtime):
            if ai._spacing_span_has_security_signal(content, span, budget):
                line = content.count("\n", 0, span[0]) + 1
                _emit(out, "AE6:spacing", rel, line, content[span[0]:span[1]][:200])
                break
        ctx_line = ai._contextual_ignorable_security_line(content, budget)
        if ctx_line is not None:
            _emit(out, "AE6:contextual", rel, ctx_line, "contextual-ignorable")
        m = next(ai._obfuscated_instruction_matches(content, budget.check_runtime), None)
        if m is not None:
            line = content.count("\n", 0, getattr(m, "evidence_offset", 0)) + 1
            _emit(out, "AE6:targeted", rel, line, "obfuscated-instruction")
    except Exception as exc:
        _emit(out, "AE6", rel, 0, f"ANALYZER_ERROR:{type(exc).__name__}")


def _walk_files(roots):
    for root in roots:
        root = Path(root)
        for p in sorted(root.rglob("*")):
            if p.is_file() and not p.name.startswith("."):
                yield root, p


def _run_bundle(root, bundle, out):
    """State rules: construct SkillspectorState fields for one bundle dir."""
    bundle = Path(bundle)
    files = {}
    for p in sorted(bundle.rglob("*")):
        if p.is_file():
            rel = str(PurePosixPath(p.relative_to(bundle)))
            try:
                files[rel] = p.read_text(encoding="utf-8")
            except (UnicodeDecodeError, ValueError):
                pass
    state = {
        "components": set(files),
        "local_file_cache": files,
        "file_cache": files,
        "artifact_inventory": [],
    }
    try:
        resp = bundled_execution_surface.node(state)
        for f in resp.get("findings", []):
            rid = getattr(f, "rule_id", "")
            _emit(out, rid, f"{bundle.name}/{getattr(f, 'file', '')}",
                  getattr(f, "start_line", 1), getattr(f, "matched_text", "") or getattr(f, "message", ""))
    except Exception as exc:
        _emit(out, "BH2", bundle.name, 0, f"ANALYZER_ERROR:{type(exc).__name__}")
    try:
        ctx = extract_structured_skill_context(bundle)
        if isinstance(ctx, dict):
            resp = structured_skill_roles.node({"structured_skill_context": ctx})
            for s in resp.get("structured_summaries", []):
                _emit(out, "SSR-1", f"{bundle.name}/{s.get('file', '')}", 1, s.get("message", ""))
    except Exception as exc:
        _emit(out, "SSR-1", bundle.name, 0, f"ANALYZER_ERROR:{type(exc).__name__}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--corpus-root", action="append", default=[])
    ap.add_argument("--bundle-root", action="append", default=[])
    args = ap.parse_args()

    out = []
    for root, path in _walk_files(args.corpus_root):
        rel = str(PurePosixPath(path.relative_to(root)))
        try:
            content = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, ValueError):
            continue
        suffix = path.suffix.lower()
        ftype = FILE_TYPES.get(suffix, "other")
        _run_file_rules(content, rel, ftype, out)
        _run_tp1(content, rel, out)
        _run_p9(content, rel, ftype, out)
        _run_ae6_lanes(content, rel, out)

    for root in args.bundle_root:
        root = Path(root)
        for child in sorted(root.iterdir()):
            if child.is_dir():
                _run_bundle(root, child, out)

    for rec in out:
        print(json.dumps(rec, ensure_ascii=False))


if __name__ == "__main__":
    main()
