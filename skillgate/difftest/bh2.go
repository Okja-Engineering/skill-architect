package difftest

// SK-P-BH2 — port of bundled_execution_surface's BH2 closed proof:
// hook declarations over hooks/hooks.json and .claude/settings*.json,
// then _bh2_proof (http handler on sensitive event w/ remote URL, or
// command handler curl/wget/scp exfil proofs).
//
// Declared gaps vs upstream: JSON duplicate-key rejection, `if`-rule
// validation (handlers carrying "if" are skipped), WHATWG URL semantics
// (net/url approximates pywhatwgurl), punycode label validation, and the
// bidi hostname rule. The differ measures any resulting divergence.

import (
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const bh2MaxFileChars = 1_000_000
const bh2MaxDeclChars = 16_384

var bh2ApplicablePaths = map[string]bool{
	"hooks/hooks.json":            true,
	".claude/settings.json":       true,
	".claude/settings.local.json": true,
}

var bh2SensitiveEvents = map[string]bool{
	"UserPromptSubmit": true, "UserPromptExpansion": true, "MessageDisplay": true,
	"PreToolUse": true, "PermissionRequest": true, "PermissionDenied": true,
	"PostToolUse": true, "PostToolUseFailure": true, "PostToolBatch": true,
	"SubagentStop": true, "TaskCreated": true, "TaskCompleted": true,
	"Stop": true, "StopFailure": true, "PreCompact": true, "PostCompact": true,
	"Elicitation": true, "ElicitationResult": true,
}

var bh2SensitiveDirSuffixes = []string{"/.ssh", "/.aws", "/.kube", "/.config/gcloud"}

var bh2SensitiveFileSuffixes = map[string]bool{
	"/.claude/settings.json": true, "/.claude/settings.local.json": true,
	"/.claude/.credentials.json": true, "/.docker/config.json": true,
	"/.netrc": true, "/.npmrc": true,
}

var bh2KnownEvents = map[string]bool{
	"ConfigChange": true, "CwdChanged": true, "DirectoryAdded": true,
	"Elicitation": true, "ElicitationResult": true, "FileChanged": true,
	"InstructionsLoaded": true, "MessageDisplay": true, "Notification": true,
	"PermissionDenied": true, "PermissionRequest": true, "PostCompact": true,
	"PostToolBatch": true, "PostToolUse": true, "PostToolUseFailure": true,
	"PreCompact": true, "PreToolUse": true, "SessionEnd": true,
	"SessionStart": true, "Setup": true, "Stop": true, "StopFailure": true,
	"SubagentStart": true, "SubagentStop": true, "TaskCompleted": true,
	"TaskCreated": true, "TeammateIdle": true, "UserPromptExpansion": true,
	"UserPromptSubmit": true, "WorktreeCreate": true, "WorktreeRemove": true,
}

var bh2NoMatcherEvents = map[string]bool{
	"CwdChanged": true, "MessageDisplay": true, "PostToolBatch": true,
	"Stop": true, "TaskCompleted": true, "TaskCreated": true,
	"TeammateIdle": true, "UserPromptSubmit": true, "WorktreeCreate": true,
	"WorktreeRemove": true,
}

var bh2CommandMCPOnlyEvents = map[string]bool{"SessionStart": true, "Setup": true}

var bh2PromptAgentEvents = map[string]bool{
	"PermissionDenied": true, "PermissionRequest": true, "PostToolBatch": true,
	"PostToolUse": true, "PostToolUseFailure": true, "PreToolUse": true,
	"Stop": true, "SubagentStop": true, "TaskCompleted": true,
	"TaskCreated": true, "TeammateIdle": true, "UserPromptExpansion": true,
	"UserPromptSubmit": true,
}

var bh2KnownHandlerTypes = map[string]bool{
	"command": true, "http": true, "mcp_tool": true, "prompt": true, "agent": true,
}

var (
	bh2AbsoluteHomePathRE = regexp.MustCompile(`^/(?:Users/([^/]+)|home/([^/]+)|root)(/.*)$`)
	bh2RemotePathRE       = regexp.MustCompile(`^(?:[A-Za-z0-9._-]+@)?([A-Za-z0-9.-]+):([^\s]+)$`)
	bh2BracketedRemoteRE  = regexp.MustCompile(`^(?:[A-Za-z0-9._-]+@)?\[([0-9A-Fa-f:.]+)\]:([^\s]+)$`)
	bh2CurlHTTPURL        = regexp.MustCompile(`(?i)^https?:/{1,3}[^/]`)
	bh2WgetHTTPURL        = regexp.MustCompile(`(?i)^https?://[^/]`)
	bh2HeaderNameRE       = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)
	bh2DrivePrefixRE      = regexp.MustCompile(`^[A-Za-z]:`)
	bh2HostLabelRE        = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
	bh2NumericHostRE      = regexp.MustCompile(`^[0-9.]+$`)
	bh2ScpUserRE          = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func bh2SafeLiteral(v string) bool {
	if len(v) > bh2MaxDeclChars || strings.ContainsRune(v, 0) {
		return false
	}
	for _, r := range v {
		if r >= 0xD800 && r <= 0xDFFF {
			return false
		}
	}
	return true
}

// _is_non_remote_host — localhost names + IP literals that are loopback,
// unspecified, multicast, or 255.255.255.255. inet_aton shorthands are
// approximated (dotted forms 1–4 parts, hex/octal single ints).
func bh2IsNonRemoteHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSuffix(host, "."))
	if strings.HasPrefix(normalized, "[") && strings.HasSuffix(normalized, "]") {
		normalized = normalized[1 : len(normalized)-1]
	}
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return true
	}
	ip := net.ParseIP(normalized)
	if ip == nil {
		ip = inetAton(normalized)
	}
	if ip == nil {
		return false
	}
	return addrIsNonRemote(ip)
}

func addrIsNonRemote(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		return ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
			v4[0] == 255 && v4[1] == 255 && v4[2] == 255 && v4[3] == 255
	}
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast()
}

// inetAton approximates socket.inet_aton: 1–4 dotted numeric parts,
// each decimal/octal/hex, with the last part filling remaining bytes.
func inetAton(s string) net.IP {
	if strings.ContainsAny(s, "xX") && !strings.HasPrefix(strings.ToLower(s), "0x") {
		return nil
	}
	parts := strings.Split(s, ".")
	if len(parts) > 4 || len(parts) == 0 {
		return nil
	}
	var nums []uint64
	for _, p := range parts {
		if p == "" {
			return nil
		}
		n, err := strconv.ParseUint(p, 0, 32)
		if err != nil {
			return nil
		}
		nums = append(nums, n)
	}
	if len(nums) == 1 {
		if nums[0] > 0xFFFFFFFF {
			return nil
		}
		return net.IPv4(byte(nums[0]>>24), byte(nums[0]>>16), byte(nums[0]>>8), byte(nums[0]))
	}
	last := nums[len(nums)-1]
	head := nums[:len(nums)-1]
	for _, n := range head {
		if n > 255 {
			return nil
		}
	}
	maxLast := uint64(1) << uint(8*(5-len(nums)))
	if last >= maxLast {
		return nil
	}
	b := make([]byte, 4)
	for i, n := range head {
		b[i] = byte(n)
	}
	shift := uint(8 * (4 - len(head) - 1))
	for i := len(head); i < 4; i++ {
		b[i] = byte(last >> shift)
		shift -= 8
	}
	return net.IPv4(b[0], b[1], b[2], b[3])
}

// _is_valid_literal_host (punycode label validation is a declared gap).
func bh2IsValidLiteralHost(host string) bool {
	if strings.HasSuffix(host, "..") {
		return false
	}
	normalized := strings.ToLower(strings.TrimSuffix(host, "."))
	if normalized == "" {
		return false
	}
	for _, r := range normalized {
		if r > 127 {
			return false
		}
	}
	if strings.Contains(normalized, "%") {
		return false
	}
	if net.ParseIP(normalized) != nil || inetAton(normalized) != nil {
		return true
	}
	if strings.HasPrefix(normalized, "0x") || bh2NumericHostRE.MatchString(normalized) {
		return false
	}
	for _, label := range strings.Split(normalized, ".") {
		if len(label) < 1 || len(label) > 63 || !bh2HostLabelRE.MatchString(label) {
			return false
		}
	}
	return true
}

// _parse_http_url approximation: net/url + http/https + hostname + no '$'.
// WHATWG-only normalizations and the bidi hostname rule are declared gaps.
func bh2ParseHTTPURL(v any) *url.URL {
	s, ok := v.(string)
	if !ok || len(s) > bh2MaxDeclChars {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" {
		return nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil
	}
	if strings.Contains(u.Hostname(), "$") {
		return nil
	}
	return u
}

func bh2IsExternalHTTPURL(v any) bool {
	s, ok := v.(string)
	if !ok || !bh2SafeLiteral(s) {
		return false
	}
	if strings.Contains(s, "\\") {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == 0x85 || r == 0xA0 {
			return false
		}
	}
	u := bh2ParseHTTPURL(v)
	return u != nil && !bh2IsNonRemoteHost(u.Hostname())
}

// _is_remote_http: parseable http(s) URL, remote host, sendable headers.
func bh2IsRemoteHTTP(handler map[string]any) bool {
	headers, _ := handler["headers"].(map[string]any)
	u := bh2ParseHTTPURL(handler["url"])
	if u == nil || bh2IsNonRemoteHost(u.Hostname()) {
		return false
	}
	for name := range headers {
		if !bh2HeaderNameRE.MatchString(name) {
			return false
		}
	}
	return true
}

// ---- sensitive paths -------------------------------------------------------

func bh2SensitiveSuffix(suffix string) bool {
	if !bh2SafeLiteral(suffix) || !strings.HasPrefix(suffix, "/") {
		return false
	}
	segments := strings.Split(suffix[1:], "/")
	if len(segments) > 0 && segments[len(segments)-1] == "" {
		segments = segments[:len(segments)-1]
	}
	if len(segments) == 0 {
		return false
	}
	for _, s := range segments {
		if s == "" || s == "." || s == ".." {
			return false
		}
	}
	if bh2SensitiveFileSuffixes[suffix] {
		return true
	}
	for _, dir := range bh2SensitiveDirSuffixes {
		if suffix == dir || strings.HasPrefix(suffix, dir+"/") {
			return true
		}
	}
	return false
}

func bh2SensitiveFileSuffix(suffix string) bool {
	if strings.HasSuffix(suffix, "/") {
		return false
	}
	for _, d := range bh2SensitiveDirSuffixes {
		if suffix == d {
			return false
		}
	}
	return bh2SensitiveSuffix(suffix)
}

func bh2IsSensitiveAbsolutePath(v any) bool {
	s, ok := v.(string)
	if !ok || !bh2SafeLiteral(s) {
		return false
	}
	m := bh2AbsoluteHomePathRE.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	// (?!\.{1,2}/): user segment must not be . or ..
	seg := m[1]
	if seg == "" {
		seg = m[2]
	}
	if seg == "." || seg == ".." {
		return false
	}
	return bh2SensitiveFileSuffix(m[3])
}

func bh2IsSensitiveShellPath(v string) bool {
	if !bh2SafeLiteral(v) {
		return false
	}
	for _, anchor := range []string{"$HOME", "${HOME}"} {
		if strings.HasPrefix(v, anchor+"/") {
			suffix := v[len(anchor):]
			if strings.ContainsAny(suffix, "\\$*?[]{}!") {
				return false
			}
			return bh2SensitiveFileSuffix(suffix)
		}
	}
	return false
}

// ---- remote destinations ---------------------------------------------------

func bh2IsRemoteDestination(v any, transport string) bool {
	s, ok := v.(string)
	if !ok || !bh2SafeLiteral(s) {
		return false
	}
	if strings.HasPrefix(s, "-") || bh2DrivePrefixRE.MatchString(s) ||
		strings.Contains(s, "\\") {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' {
			return false
		}
	}
	if strings.Contains(s, "://") {
		return bh2IsRemoteURIDestinationFull(s, transport)
	}
	m := bh2BracketedRemoteRE.FindStringSubmatch(s)
	if m == nil {
		m = bh2RemotePathRE.FindStringSubmatch(s)
	}
	if m == nil {
		return false
	}
	host, remotePath := m[1], m[2]
	if transport == "rsync" && strings.HasPrefix(remotePath, ":") {
		mod := strings.SplitN(remotePath[1:], "/", 2)[0]
		if mod == "" {
			return false
		}
	}
	return bh2IsValidLiteralHost(host) && !bh2IsNonRemoteHost(host)
}

// _is_remote_uri_destination.
func bh2IsRemoteURIDestinationFull(value, transport string) bool {
	if !strings.HasPrefix(value, transport+"://") {
		return false
	}
	authority := strings.SplitN(strings.SplitN(value, "://", 2)[1], "/", 2)
	auth := authority[0]
	if i := strings.IndexAny(auth, "?#"); i >= 0 {
		auth = auth[:i]
	}
	if transport == "scp" {
		last := auth
		if i := strings.LastIndex(auth, "@"); i >= 0 {
			last = auth[i+1:]
		}
		if strings.HasSuffix(last, ":") {
			return false
		}
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if transport == "scp" {
		if u.User != nil {
			un := u.User.Username()
			if un != "" && !bh2ScpUserRE.MatchString(un) {
				return false
			}
		}
		if strings.Contains(u.Path, "%") {
			return false
		}
	}
	if transport == "rsync" && strings.HasPrefix(u.Path, "//") {
		return false
	}
	// bracketed host must be a valid IPv6
	if strings.Contains(auth, "[") {
		afterAt := auth
		if i := strings.LastIndex(auth, "@"); i >= 0 {
			afterAt = auth[i+1:]
		}
		if strings.HasPrefix(afterAt, "[") {
			if ip := net.ParseIP(host); ip == nil || ip.To4() != nil {
				return false
			}
		}
	}
	if u.User != nil {
		if _, hasPw := u.User.Password(); hasPw {
			return false
		}
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Port() == "0" {
		return false
	}
	return bh2IsValidLiteralHost(host) &&
		u.Path != "" && u.Path != "/" &&
		!bh2IsNonRemoteHost(host)
}

// ---- flag/args helpers ------------------------------------------------------

func bh2LiteralArgs(handler map[string]any) []string {
	raw, ok := handler["args"].([]any)
	if !ok {
		return nil
	}
	args := make([]string, 0, len(raw))
	for _, a := range raw {
		s, ok := a.(string)
		if !ok || !bh2SafeLiteral(s) {
			return nil
		}
		args = append(args, s)
	}
	return args
}

// _flag_source: exactly one flag occurrence (separate arg or --flag=value).
func bh2FlagSource(args []string, flags map[string]bool) (string, []string, bool) {
	type hit struct {
		idx int
		val string
	}
	var matches []hit
	consumed := map[int]bool{}
	for i, arg := range args {
		if flags[arg] {
			if i+1 >= len(args) {
				return "", nil, false
			}
			matches = append(matches, hit{i, args[i+1]})
			consumed[i] = true
			consumed[i+1] = true
			continue
		}
		for flag := range flags {
			if strings.HasPrefix(flag, "--") && strings.HasPrefix(arg, flag+"=") {
				matches = append(matches, hit{i, arg[len(flag)+1:]})
				consumed[i] = true
				break
			}
		}
	}
	if len(matches) != 1 {
		return "", nil, false
	}
	var remaining []string
	for i, arg := range args {
		if !consumed[i] {
			remaining = append(remaining, arg)
		}
	}
	return matches[0].val, remaining, true
}

func bh2WithoutCurlTransportFlags(args []string) ([]string, bool) {
	var remaining []string
	sawSilent, sawRequest := false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-s" {
			if sawSilent {
				return nil, false
			}
			sawSilent = true
			continue
		}
		if arg == "-X" {
			if sawRequest || i+1 >= len(args) || args[i+1] != "POST" {
				return nil, false
			}
			sawRequest = true
			i++
			continue
		}
		remaining = append(remaining, arg)
	}
	return remaining, true
}

func bh2CurlURLHasUnsupportedGlob(value string) bool {
	if strings.ContainsAny(value, "{}") {
		return true
	}
	if !strings.Contains(value, "[") && !strings.Contains(value, "]") {
		return false
	}
	if !bh2CurlHTTPURL.MatchString(value) {
		return true
	}
	u := bh2ParseHTTPURL(value)
	rest := strings.SplitN(value, ":", 2)
	if len(rest) != 2 {
		return true
	}
	authPart := strings.TrimLeft(rest[1], "/")
	auth := strings.SplitN(authPart, "/", 2)[0]
	if i := strings.IndexAny(auth, "?#"); i >= 0 {
		auth = auth[:i]
	}
	return !(strings.Count(value, "[") == 1 && strings.Count(value, "]") == 1 &&
		strings.Count(auth, "[") == 1 && strings.Count(auth, "]") == 1 &&
		u != nil && strings.Contains(u.Hostname(), ":"))
}

func bh2OneExternalCurlURL(values []string) bool {
	if len(values) != 1 || !bh2CurlHTTPURL.MatchString(values[0]) {
		return false
	}
	u := bh2ParseHTTPURL(values[0])
	if u == nil || bh2CurlURLHasUnsupportedGlob(values[0]) {
		return false
	}
	if strings.ContainsAny(u.Hostname(), "!$&'()*+,;=") {
		return false
	}
	return bh2IsExternalHTTPURL(values[0])
}

func bh2OneExternalURL(values []string) bool {
	return len(values) == 1 &&
		bh2WgetHTTPURL.MatchString(values[0]) &&
		bh2IsExternalHTTPURL(values[0])
}

// ---- proofs ------------------------------------------------------------------

var bh2CurlDataFlags = map[string]bool{"-d": true, "--data": true, "--data-ascii": true, "--data-binary": true}
var bh2CurlUploadFlags = map[string]bool{"-T": true, "--upload-file": true}
var bh2WgetBodyFlags = map[string]bool{"--post-file": true, "--body-file": true}

func bh2CurlExecProof(event string, args []string) bool {
	if source, remaining, ok := bh2FlagSource(args, bh2CurlDataFlags); ok {
		urlArgs, ok := bh2WithoutCurlTransportFlags(remaining)
		if !ok {
			return false
		}
		sensitive := (source == "@-" && bh2SensitiveEvents[event]) ||
			(strings.HasPrefix(source, "@") && bh2IsSensitiveAbsolutePath(source[1:]))
		return sensitive && bh2OneExternalCurlURL(urlArgs)
	}
	source, remaining, ok := bh2FlagSource(args, bh2CurlUploadFlags)
	if !ok {
		return false
	}
	urlArgs, ok := bh2WithoutCurlTransportFlags(remaining)
	if !ok {
		return false
	}
	sensitive := (source == "-" && bh2SensitiveEvents[event]) ||
		bh2IsSensitiveAbsolutePath(source)
	return sensitive && bh2OneExternalCurlURL(urlArgs)
}

func bh2WgetExecProof(args []string) bool {
	source, remaining, ok := bh2FlagSource(args, bh2WgetBodyFlags)
	if !ok {
		return false
	}
	return bh2IsSensitiveAbsolutePath(source) && bh2OneExternalURL(remaining)
}

func bh2ShellCurlProof(event, command string) bool {
	if !bh2SafeLiteral(command) ||
		strings.ContainsAny(command, "\t\r\n\"'|&;<>`()\\*?[]!") {
		return false
	}
	tokens := strings.Split(command, " ")
	for _, tk := range tokens {
		if tk == "" {
			return false
		}
	}
	if len(tokens) == 0 || tokens[0] != "curl" {
		return false
	}
	source, remaining, ok := bh2FlagSource(tokens[1:], map[string]bool{"-d": true})
	if !ok {
		return false
	}
	urlArgs, ok := bh2WithoutCurlTransportFlags(remaining)
	if !ok {
		return false
	}
	sensitive := (source == "@-" && bh2SensitiveEvents[event]) ||
		(strings.HasPrefix(source, "@") && bh2IsSensitiveShellPath(source[1:]))
	return sensitive && bh2OneExternalCurlURL(urlArgs)
}

func bh2CommandTransport(decl bh2Decl, args []string) string {
	command, _ := decl.handler["command"].(string)
	switch command {
	case "curl":
		if bh2CurlExecProof(decl.event, args) {
			return "curl"
		}
	case "wget":
		if bh2WgetExecProof(args) {
			return "wget"
		}
	case "scp", "rsync":
		if len(args) == 2 &&
			bh2IsSensitiveAbsolutePath(args[0]) &&
			bh2IsRemoteDestination(args[1], command) {
			return command
		}
	}
	return ""
}

type bh2Decl struct {
	event       string
	handlerType string
	remoteHTTP  bool
	handler     map[string]any
}

func bh2CommandProof(decl bh2Decl) (string, bool) {
	command, ok := decl.handler["command"].(string)
	if !ok || !bh2SafeLiteral(command) {
		return "", false
	}
	if _, hasArgs := decl.handler["args"]; !hasArgs {
		shell, _ := decl.handler["shell"].(string)
		if sh, present := decl.handler["shell"]; present && sh != nil && shell != "bash" {
			return "", false
		}
		if bh2ShellCurlProof(decl.event, command) {
			return "curl", true
		}
		return "", false
	}
	args := bh2LiteralArgs(decl.handler)
	if args == nil {
		return "", false
	}
	if t := bh2CommandTransport(decl, args); t != "" {
		return t, true
	}
	return "", false
}

func bh2Proof(decl bh2Decl) (string, string, bool) {
	switch decl.handlerType {
	case "http":
		if bh2SensitiveEvents[decl.event] && decl.remoteHTTP {
			return "event_http_body", "http", true
		}
	case "command":
		if transport, ok := bh2CommandProof(decl); ok {
			return "direct_command_upload", transport, true
		}
	}
	return "", "", false
}

// ---- declaration scan ---------------------------------------------------------

func bh2HandlerStringsBounded(handler map[string]any) bool {
	for _, key := range []string{"type", "command", "url", "prompt", "server", "tool", "shell", "if"} {
		if v, ok := handler[key]; ok {
			s, isStr := v.(string)
			if !isStr {
				continue
			}
			if key == "url" && handler["type"] == "http" {
				if len(s) > bh2MaxDeclChars {
					return false
				}
			} else if !bh2SafeLiteral(s) {
				return false
			}
		}
	}
	if t, _ := handler["type"].(string); t != "command" {
		return true
	}
	raw, present := handler["args"]
	if !present {
		return true
	}
	list, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, a := range list {
		s, ok := a.(string)
		if !ok || !bh2SafeLiteral(s) {
			return false
		}
	}
	return true
}

func bh2HandlerTypeSupported(event, handlerType string) bool {
	if !bh2KnownEvents[event] || !bh2KnownHandlerTypes[handlerType] {
		return true
	}
	if bh2CommandMCPOnlyEvents[event] {
		return handlerType == "command" || handlerType == "mcp_tool"
	}
	if handlerType == "prompt" || handlerType == "agent" {
		return bh2PromptAgentEvents[event]
	}
	return true
}

func bh2HandlerShapeValid(handlerType string, handler map[string]any) bool {
	required := map[string][]string{
		"command": {"command"}, "http": {"url"}, "prompt": {"prompt"},
		"agent": {"prompt"}, "mcp_tool": {"server", "tool"},
	}[handlerType]
	if handlerType == "" {
		return false
	}
	if handlerType == "command" {
		if sh, ok := handler["shell"]; ok {
			s, isStr := sh.(string)
			if !isStr || (s != "bash" && s != "powershell") {
				return false
			}
		}
	}
	for _, field := range required {
		v, present := handler[field]
		if !present {
			return false
		}
		if handlerType == "http" && field == "url" {
			s, isStr := v.(string)
			if !isStr || len(s) > bh2MaxDeclChars {
				return false
			}
		} else {
			s, isStr := v.(string)
			if !isStr || !bh2SafeLiteral(s) {
				return false
			}
		}
	}
	if handlerType == "http" {
		if bh2ParseHTTPURL(handler["url"]) == nil {
			return false
		}
	}
	return true
}

func bh2HookDeclaration(event string, raw any) (bh2Decl, bool) {
	handler, ok := raw.(map[string]any)
	if !ok || !bh2HandlerStringsBounded(handler) {
		return bh2Decl{}, false
	}
	rawType, ok := handler["type"].(string)
	if !ok || !bh2HandlerShapeValid(rawType, handler) || !bh2HandlerTypeSupported(event, rawType) {
		return bh2Decl{}, false
	}
	if _, hasIf := handler["if"]; hasIf {
		return bh2Decl{}, false // declared gap: `if` rule validation not ported
	}
	return bh2Decl{
		event:       event,
		handlerType: rawType,
		remoteHTTP:  rawType == "http" && bh2IsRemoteHTTP(handler),
		handler:     handler,
	}, true
}

// scanBH2 mirrors _scan_declarations' hooks leg (permissions are a separate
// rule — BH3 — out of this port's scope).
func scanBH2(document map[string]any) []bh2Decl {
	hooks, ok := document["hooks"].(map[string]any)
	if !ok {
		return nil
	}
	var decls []bh2Decl
	for event, rawGroups := range hooks {
		if !bh2SafeLiteral(event) {
			continue
		}
		groups, ok := rawGroups.([]any)
		if !ok {
			continue
		}
		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				continue
			}
			handlers, ok := group["hooks"].([]any)
			if !ok {
				continue
			}
			for _, h := range handlers {
				if d, ok := bh2HookDeclaration(event, h); ok {
					decls = append(decls, d)
				}
			}
		}
	}
	return decls
}

// DetectBH2 runs the port over one bundle (relpath → content map).
// Emits one BH2 record per applicable path carrying at least one proof.
func DetectBH2(bundle string, files map[string]string) []Match {
	var out []Match
	for path, content := range files {
		if !bh2ApplicablePaths[path] || len(content) > bh2MaxFileChars {
			continue
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(content), &document); err != nil {
			continue // opaque content → upstream records a partial event, no finding
		}
		var proofs []string
		for _, decl := range scanBH2(document) {
			if _, transport, ok := bh2Proof(decl); ok {
				proofs = append(proofs, transport)
			}
		}
		if len(proofs) > 0 {
			out = append(out, Match{
				Rule: "BH2", File: bundle + "/" + path, Line: 1,
				Evidence: "proofs:" + strings.Join(proofs, ","),
			})
		}
	}
	return out
}

// DetectSSR1 ports the SSR-1 gate: a *.aisop.json candidate whose payload is
// [system(protocol AISOP V*/AISP V*), user] with aisop/aisp_contract carrying
// functions or resources.
func DetectSSR1(bundle string, files map[string]string) []Match {
	var out []Match
	for path, content := range files {
		lower := strings.ToLower(path)
		if !strings.HasSuffix(lower, ".aisop.json") {
			continue
		}
		parts := strings.Split(path, "/")
		skip := false
		for _, p := range parts[:len(parts)-1] {
			switch p {
			case ".git", "__pycache__", "node_modules", ".venv", "venv", ".tox", ".pytest_cache":
				skip = true
			}
			if strings.HasPrefix(p, ".") && p != ".aisop" {
				skip = true
			}
		}
		if skip {
			continue
		}
		var payload []map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil || len(payload) != 2 {
			continue
		}
		sys, usr := payload[0], payload[1]
		if sys["role"] != "system" || usr["role"] != "user" {
			continue
		}
		sysContent, ok := sys["content"].(map[string]any)
		if !ok {
			continue
		}
		usrContent, ok := usr["content"].(map[string]any)
		if !ok {
			continue
		}
		protocol, ok := sysContent["protocol"].(string)
		if !ok || (!strings.HasPrefix(protocol, "AISOP V") && !strings.HasPrefix(protocol, "AISP V")) {
			continue
		}
		aisop, _ := usrContent["aisop"].(map[string]any)
		aisp, _ := usrContent["aisp_contract"].(map[string]any)
		if aisop == nil && aisp == nil {
			continue
		}
		// function_names / resource_anchors must be non-empty: any "functions"
		// list with a named entry or "resources" map suffices for the gate.
		hasFns := func(m map[string]any) bool {
			if m == nil {
				return false
			}
			if fns, ok := m["functions"].([]any); ok && len(fns) > 0 {
				return true
			}
			return false
		}
		hasRes := false
		if aisp != nil {
			if res, ok := aisp["resources"].(map[string]any); ok && len(res) > 0 {
				hasRes = true
			}
		}
		if !hasFns(usrContent) && !hasFns(aisop) && !hasFns(aisp) && !hasRes {
			continue
		}
		out = append(out, Match{
			Rule: "SSR-1", File: bundle + "/" + path, Line: 1,
			Evidence: "Structured " + strings.Split(protocol, " ")[0] + " bundle",
		})
	}
	return out
}
