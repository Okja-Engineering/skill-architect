package difftest

// The 15 sampled rules: SK-P* ids, upstream Origin (Apache-2.0, NVIDIA
// SkillSpector @ 8421a2e), and the record key each detector emits.

// FileRules run per document.
var FileRules = []PortedRule{
	{SKID: "SK-P-EA1", Origin: "SkillSpector EA1", Upstream: "EA1", Detect: detectEA1},
	{SKID: "SK-P-P5", Origin: "SkillSpector P5", Upstream: "P5", Detect: detectP5},
	{SKID: "SK-P-MP2", Origin: "SkillSpector MP2", Upstream: "MP2", Detect: detectMP2},
	{SKID: "SK-P-OH1", Origin: "SkillSpector OH1 (regex+fallback lanes)", Upstream: "OH1",
		Detect: detectOH1, Note: "AST lane ceded: python-file findings are upstream-only"},
	{SKID: "SK-P-TM1", Origin: "SkillSpector TM1 (regex lane)", Upstream: "TM1",
		Detect: detectTM1, Note: "shell-lexer lane ceded: upstream-only"},
	{SKID: "SK-P-RA1", Origin: "SkillSpector RA1", Upstream: "RA1", Detect: detectRA1},
	{SKID: "SK-P-P6", Origin: "SkillSpector P6", Upstream: "P6", Detect: detectP6},
	{SKID: "SK-P-TP1", Origin: "SkillSpector TP1 (base64 lane)", Upstream: "TP1:base64",
		Detect: detectTP1Base64, Note: "other TP1 lanes (data-uri/comments/zero-width) not ported"},
	{SKID: "SK-P-P9", Origin: "SkillSpector P9 whitespace_padding", Upstream: "P9", Detect: detectP9},
	{SKID: "SK-P-AE6", Origin: "SkillSpector AE6 (letter-spacing lane)", Upstream: "AE6:spacing",
		Detect: detectAE6Spacing, Note: "contextual-ignorable + targeted-instruction lanes not ported"},
	{SKID: "SK-P-AS3", Origin: "SkillSpector AS3", Upstream: "AS3", Detect: detectAS3},
	{SKID: "SK-P-AR2", Origin: "SkillSpector AR2", Upstream: "AR2",
		Detect: detectAR2, Note: "evidence is the directive line (declared rewrite)"},
}

// BundleRules run per bundle directory (relpath → content).
var BundleRules = []struct {
	SKID     string
	Origin   string
	Upstream string
	Detect   func(bundle string, files map[string]string) []Match
	Note     string
}{
	{SKID: "SK-P-BH2", Origin: "SkillSpector BH2 (bundled_execution_surface)", Upstream: "BH2",
		Detect: DetectBH2, Note: "BH1/BH3 lanes and `if`-rule/punycode/bidi corners are declared gaps"},
	{SKID: "SK-P-SSR-1", Origin: "SkillSpector SSR-1 (structured_skill_roles)", Upstream: "SSR-1",
		Detect: DetectSSR1},
}

// CededRule documents a rule with no port.
var CededRule = PortedRule{
	SKID: "SK-P-TT2", Origin: "SkillSpector TT2 (behavioral_taint_tracking)",
	Upstream: "TT2", Detect: nil,
	Note: "ceded — Python-AST taint flow has no Go equivalent in scope",
}
