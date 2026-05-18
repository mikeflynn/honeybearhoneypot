# Network Apps & Editor Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add config-driven `curl` and `nmap` honeypot commands; split today's `cat` into a streaming `cat` and a paged `vi`; keep `less`/`more` paged.

**Architecture:** New `Exec` funcs in `internal/honeypot/filesystem/exec.go`. Curl/nmap data structs live in the `filesystem` package (the `config` package imports `filesystem`, so the cycle forbids the reverse). `main.go` calls new setter functions in `filesystem/extra.go` after parsing config. The paged-viewport state machine in `model.go` is renamed from `"cat"` → `"vi"` since that's now the only command that uses the viewport via help+less+more+vi (help already piggybacks on this state).

**Tech Stack:** Go 1.24, Charmbracelet Bubble Tea v2, existing internal packages only.

**Note for executors:** Per `CLAUDE.md`, do **not** run `git commit`. Stop after each task is implemented and tests pass; the maintainer commits manually.

**Spec:** `docs/superpowers/specs/2026-05-17-network-apps-design.md`

---

## File Structure

Files this plan touches:

- `internal/honeypot/filesystem/extra.go` — Add `CurlResponse`, `NmapPort`, `NmapHost` types and `SetCurlResponses` / `SetNmapHosts` setters. Add package-level `curlResponses` / `nmapHosts` slices.
- `internal/honeypot/filesystem/exec.go` — Add `curlExec`, `nmapExec`, `viExec`; rewrite `catExec`. Add `normalizeURL` and `stripCurlFlags` / `stripNmapFlags` helpers.
- `internal/honeypot/filesystem/filesystem.go` — Add `vi`, `curl`, `nmap` nodes under `/usr/bin`; point existing `cat` at the new streaming `catExec`; point `less`/`more` at `viExec`.
- `internal/honeypot/filesystem/exec_test.go` — New file with unit tests for the four execs.
- `internal/honeypot/model.go` — Rename `"cat"` → `"vi"` in the `Update` switch (line 264) and the `View` branch (line 298). Update footer hint string.
- `internal/honeypot/filesystem/filesystem.go` (separate touch) — Rename `SetRunningCmd("cat")` → `SetRunningCmd("vi")` inside `helpExec`-equivalent inline closure at line 374.
- `internal/config/config.go` — Add `CurlResponses` and `NmapHosts` fields (typed from the `filesystem` package) plus merge clauses.
- `main.go` — Call `filesystem.SetCurlResponses(cfg.CurlResponses)` and `filesystem.SetNmapHosts(cfg.NmapHosts)` near the existing `SetAdditionalNodes` call.
- `config.sample.json` — Add `curl_responses` and `nmap_hosts` examples.

---

## Task 1: Add curl/nmap data types and setters in filesystem package

**Files:**
- Modify: `internal/honeypot/filesystem/extra.go`

- [ ] **Step 1: Append types, package vars, and setters**

Edit `internal/honeypot/filesystem/extra.go`. After the existing `noFun` declaration and before `addNode`, add:

```go
// CurlResponse is a single fake URL-to-body mapping consumed by curlExec.
type CurlResponse struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

// NmapPort is a single open-port entry for a fake host.
type NmapPort struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
	Version string `json:"version,omitempty"`
}

// NmapHost is a single fake host with its open ports.
type NmapHost struct {
	IP    string     `json:"ip"`
	Ports []NmapPort `json:"ports"`
}

var (
	curlResponses []CurlResponse
	nmapHosts     []NmapHost
)

// SetCurlResponses installs the curl response table used by curlExec.
func SetCurlResponses(responses []CurlResponse) {
	curlResponses = responses
}

// SetNmapHosts installs the nmap host table used by nmapExec.
func SetNmapHosts(hosts []NmapHost) {
	nmapHosts = hosts
}

// CurlResponses returns the installed curl responses (test helper / accessor).
func CurlResponses() []CurlResponse {
	return curlResponses
}

// NmapHosts returns the installed nmap hosts (test helper / accessor).
func NmapHosts() []NmapHost {
	return nmapHosts
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/honeypot/filesystem/`
Expected: no output, exit 0.

- [ ] **Step 3: Stop for review**

Do not commit. Hand back to maintainer.

---

## Task 2: Wire config to the new setters

**Files:**
- Modify: `internal/config/config.go`
- Modify: `main.go`

- [ ] **Step 1: Add config fields**

In `internal/config/config.go`, inside the `Config` struct (after `ExportTypes`), add:

```go
	CurlResponses []filesystem.CurlResponse `json:"curl_responses,omitempty"`
	NmapHosts     []filesystem.NmapHost     `json:"nmap_hosts,omitempty"`
```

- [ ] **Step 2: Add merge clauses**

In the same file's `merge()` function, after the `ExportTypes` clause, add:

```go
	if len(src.CurlResponses) > 0 {
		dst.CurlResponses = src.CurlResponses
	}
	if len(src.NmapHosts) > 0 {
		dst.NmapHosts = src.NmapHosts
	}
```

- [ ] **Step 3: Call setters from main.go**

In `main.go`, immediately after the existing `filesystem.SetAdditionalNodes(cfg.Filesystem)` line (currently line 32), insert:

```go
	filesystem.SetCurlResponses(cfg.CurlResponses)
	filesystem.SetNmapHosts(cfg.NmapHosts)
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 5: Stop for review**

---

## Task 3: Rename the `"cat"` viewport state to `"vi"`

This decouples the model's paged viewport branch from the soon-to-be-streaming cat command. `help`, `less`, `more`, and the future `vi` all share this branch.

**Files:**
- Modify: `internal/honeypot/model.go` (lines 264 and 298)
- Modify: `internal/honeypot/filesystem/exec.go` (line 288)
- Modify: `internal/honeypot/filesystem/filesystem.go` (line 374)

- [ ] **Step 1: Update model.go Update switch**

Replace line 264 of `internal/honeypot/model.go`:

```go
		case "cat":
```

with:

```go
		case "vi":
```

- [ ] **Step 2: Update model.go View branch and footer hint**

In `internal/honeypot/model.go` around line 298, replace:

```go
	if m.runningCommand == "cat" && m.viewportReady {
		m.viewport.SetHeight(m.height - footerHeight)

		return tea.NewView("" +
			m.viewport.View() +
			"\n" +
			m.quitStyle.Render("ctrl + c to exit this file.\n"))
```

with:

```go
	if m.runningCommand == "vi" && m.viewportReady {
		m.viewport.SetHeight(m.height - footerHeight)

		return tea.NewView("" +
			m.viewport.View() +
			"\n" +
			m.quitStyle.Render(":q or ctrl + c to exit this file.\n"))
```

- [ ] **Step 3: Update filesystem/exec.go catExec setter**

In `internal/honeypot/filesystem/exec.go` line 288, change:

```go
		return SetRunningCmd("cat")
```

to:

```go
		return SetRunningCmd("vi")
```

(This is in the existing `catExec`, which becomes `viExec` in Task 5. Renaming the literal now keeps the model in sync while subsequent tasks land.)

- [ ] **Step 4: Update filesystem/filesystem.go help command setter**

In `internal/honeypot/filesystem/filesystem.go` line 374, change:

```go
										return SetRunningCmd("cat")
```

to:

```go
										return SetRunningCmd("vi")
```

- [ ] **Step 5: Build and run existing tests**

Run: `go build ./... && go test ./...`
Expected: build succeeds; existing tests pass (CGO required — if `go test` is unavailable in the worktree because of CGO/SQLite, run `go build ./...` only and note this in the handoff).

- [ ] **Step 6: Stop for review**

---

## Task 4: Add new streaming `catExec` and rename old one to `viExec`

**Files:**
- Modify: `internal/honeypot/filesystem/exec.go` (around lines 285-311)

- [ ] **Step 1: Rename existing catExec to viExec and add status line**

Replace the existing `catExec` block (lines 285-311) with two functions. First, the new `viExec` (which is the old `catExec` with a status line appended to the file body):

```go
func viExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmds := []tea.Cmd{}
	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		return SetRunningCmd("vi")
	}))

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		if len(params) == 0 {
			splash := "\n\n\n" +
				"                VIM - Vi IMproved\n" +
				"                  (read-only mode)\n\n" +
				"               type  :q  to exit\n"
			return FileContentsMsg(splash + viStatusLine("[No Name]", splash))
		}

		target, err := GetNodeByPath(dir, params[0])
		if err != nil || target == nil {
			return OutputMsg(fmt.Sprintf("E484: Can't open file %s", params[0]))
		}

		if target.IsDirectory() {
			return OutputMsg(fmt.Sprintf("E17: \"%s\" is a directory", params[0]))
		}

		fileData, err := target.Open()
		if err != nil {
			return OutputMsg("vi: " + err.Error())
		}

		body := string(fileData)
		return FileContentsMsg(body + viStatusLine(params[0], body))
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

func viStatusLine(name, body string) string {
	lines := strings.Count(body, "\n")
	if len(body) > 0 && !strings.HasSuffix(body, "\n") {
		lines++
	}
	return fmt.Sprintf("\n\"%s\" [readonly]  %dL, %dB\n", name, lines, len(body))
}
```

Then add the new streaming `catExec` right below:

```go
func catExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		if len(params) == 0 {
			return OutputMsg("")
		}

		target, err := GetNodeByPath(dir, params[0])
		if err != nil || target == nil {
			return OutputMsg(fmt.Sprintf("cat: %s: No such file or directory", params[0]))
		}

		if target.IsDirectory() {
			return OutputMsg(fmt.Sprintf("cat: %s: Is a directory", params[0]))
		}

		fileData, err := target.Open()
		if err != nil {
			return OutputMsg("cat: " + err.Error())
		}

		return OutputMsg(string(fileData))
	})
	return &cmd
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/honeypot/filesystem/`
Expected: success.

- [ ] **Step 3: Stop for review**

---

## Task 5: Register `vi` node and repoint `less` / `more`

**Files:**
- Modify: `internal/honeypot/filesystem/filesystem.go` (the `/usr/bin` children block, around lines 502-531)

- [ ] **Step 1: Update cat/less/more nodes and add vi node**

In `internal/honeypot/filesystem/filesystem.go`, locate the existing nodes for `cat` (~line 502), `less` (~line 513), and `more` (~line 523). The `cat` node already references `catExec`, which is now the streaming version (good). Change `less` and `more` to reference `viExec` instead, and insert a new `vi` node right after `more`. Replace the three contiguous nodes with this block:

```go
							{
								Name:      "cat",
								Path:      "/usr/bin/cat",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      catExec,
								HelpText:  catHelp,
							},
							{
								Name:      "less",
								Path:      "/usr/bin/less",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      viExec,
								HelpText:  catHelp,
							},
							{
								Name:      "more",
								Path:      "/usr/bin/more",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      viExec,
								HelpText:  catHelp,
							},
							{
								Name:      "vi",
								Path:      "/usr/bin/vi",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								Exec:      viExec,
								HelpText:  "Usage: vi [file]\n A read-only fake vi editor. Use :q or ctrl+c to exit.",
							},
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Stop for review**

---

## Task 6: Tests for cat (streaming) and vi (paged + status line)

**Files:**
- Create: `internal/honeypot/filesystem/exec_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/honeypot/filesystem/exec_test.go`:

```go
package filesystem

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// runExec runs an exec func and returns the produced messages in order.
// For tea.Batch results, sub-commands are executed and flattened.
func runExec(t *testing.T, exec func(*Node, []string, string, string, map[string]string) *tea.Cmd, params []string) []tea.Msg {
	t.Helper()
	Initialize()
	home, err := GetNodeByPath(SystemRoot, "home/you")
	if err != nil || home == nil {
		t.Fatalf("failed to resolve /home/you: %v", err)
	}
	cmd := exec(home, params, "you", "you", map[string]string{})
	if cmd == nil {
		return nil
	}
	return flatten((*cmd)())
}

func flatten(msg tea.Msg) []tea.Msg {
	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, flatten(c())...)
		}
		return out
	default:
		return []tea.Msg{m}
	}
}

func TestCatExecMissingFile(t *testing.T) {
	msgs := runExec(t, catExec, []string{"nope.txt"})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	out, ok := msgs[0].(OutputMsg)
	if !ok {
		t.Fatalf("expected OutputMsg, got %T", msgs[0])
	}
	if !strings.Contains(string(out), "No such file") {
		t.Errorf("expected missing-file error, got %q", string(out))
	}
}

func TestCatExecDirectory(t *testing.T) {
	msgs := runExec(t, catExec, []string{"/etc"})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	out, ok := msgs[0].(OutputMsg)
	if !ok {
		t.Fatalf("expected OutputMsg, got %T", msgs[0])
	}
	if !strings.Contains(string(out), "Is a directory") {
		t.Errorf("expected directory error, got %q", string(out))
	}
}

func TestCatExecNoArgs(t *testing.T) {
	msgs := runExec(t, catExec, nil)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	if out, ok := msgs[0].(OutputMsg); !ok || string(out) != "" {
		t.Errorf("expected empty OutputMsg, got %T %q", msgs[0], msgs[0])
	}
}

func TestCatExecHit(t *testing.T) {
	msgs := runExec(t, catExec, []string{"/home/you/patch.md"})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}
	out, ok := msgs[0].(OutputMsg)
	if !ok {
		t.Fatalf("expected OutputMsg, got %T", msgs[0])
	}
	if len(string(out)) == 0 {
		t.Error("expected non-empty file contents")
	}
}

func TestViExecSetsRunningCmd(t *testing.T) {
	msgs := runExec(t, viExec, []string{"/home/you/patch.md"})
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 msgs, got %d", len(msgs))
	}
	rc, ok := msgs[0].(SetRunningCmd)
	if !ok || string(rc) != "vi" {
		t.Errorf("expected SetRunningCmd(\"vi\"), got %T %v", msgs[0], msgs[0])
	}
	body, ok := msgs[1].(FileContentsMsg)
	if !ok {
		t.Fatalf("expected FileContentsMsg, got %T", msgs[1])
	}
	if !strings.Contains(string(body), "[readonly]") {
		t.Errorf("expected status line, got %q", string(body))
	}
}

func TestViExecNoArgsSplash(t *testing.T) {
	msgs := runExec(t, viExec, nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 msgs, got %d", len(msgs))
	}
	body, ok := msgs[1].(FileContentsMsg)
	if !ok {
		t.Fatalf("expected FileContentsMsg, got %T", msgs[1])
	}
	if !strings.Contains(string(body), "VIM") {
		t.Errorf("expected splash text, got %q", string(body))
	}
}
```

- [ ] **Step 2: Run tests; expect pass**

Run: `go test ./internal/honeypot/filesystem/ -run "TestCat|TestVi" -v`
Expected: PASS for all six tests. (If CGO/SQLite is unavailable, `go vet ./internal/honeypot/filesystem/` is an acceptable fallback and must pass.)

- [ ] **Step 3: Stop for review**

---

## Task 7: Implement `curlExec` with helpers

**Files:**
- Modify: `internal/honeypot/filesystem/exec.go`

- [ ] **Step 1: Add helpers and curlExec**

Append to `internal/honeypot/filesystem/exec.go`:

```go
// normalizeURL lowercases scheme + host, prepends http:// if no scheme,
// and trims a trailing slash from the path. Returns ("", "") on parse failure.
// Returns (normalized, host).
func normalizeURL(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	schemeIdx := strings.Index(s, "://")
	scheme := strings.ToLower(s[:schemeIdx])
	rest := s[schemeIdx+3:]
	hostEnd := strings.IndexAny(rest, "/?#")
	host := rest
	tail := ""
	if hostEnd >= 0 {
		host = rest[:hostEnd]
		tail = rest[hostEnd:]
	}
	host = strings.ToLower(host)
	// Trim a single trailing slash on the path portion only, if no query/fragment.
	if tail == "/" {
		tail = ""
	} else if strings.HasSuffix(tail, "/") && !strings.ContainsAny(tail, "?#") {
		tail = strings.TrimSuffix(tail, "/")
	}
	return scheme + "://" + host + tail, host
}

// stripCurlFlags removes flag tokens (and the next token for value-taking flags)
// from args and returns the remaining positional args.
func stripCurlFlags(args []string) []string {
	valueTaking := map[string]bool{
		"-X": true, "-H": true, "-d": true, "-o": true,
		"-A": true, "-e": true, "-u": true,
	}
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && a != "-" {
			if valueTaking[a] && i+1 < len(args) {
				i++ // skip the value
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func curlExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		if len(params) == 0 {
			return OutputMsg("curl: try 'curl --help' or 'curl --manual' for more information")
		}

		headersOnly := hasFlag(params, "-I") || hasFlag(params, "--head")
		positional := stripCurlFlags(params)
		if len(positional) == 0 {
			return OutputMsg("curl: no URL specified!")
		}

		normalized, host := normalizeURL(positional[0])

		var body string
		var found bool
		for _, r := range curlResponses {
			rn, _ := normalizeURL(r.URL)
			if rn == normalized {
				body = r.Body
				found = true
				break
			}
		}

		if !found {
			return OutputMsg(fmt.Sprintf("curl: (6) Could not resolve host: %s", host))
		}

		headers := fmt.Sprintf("HTTP/1.1 200 OK\nContent-Type: text/html\nContent-Length: %d\n", len(body))
		if headersOnly {
			return OutputMsg(headers)
		}
		return OutputMsg(headers + "\n" + body)
	})
	return &cmd
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/honeypot/filesystem/`
Expected: success.

- [ ] **Step 3: Stop for review**

---

## Task 8: Implement `nmapExec` with helper

**Files:**
- Modify: `internal/honeypot/filesystem/exec.go`

- [ ] **Step 1: Add stripNmapFlags and nmapExec**

Append to `internal/honeypot/filesystem/exec.go`:

```go
func stripNmapFlags(args []string) []string {
	valueTaking := map[string]bool{
		"-p": true, "-oN": true, "-oX": true, "-oG": true,
		"-iL": true, "-e": true, "-S": true,
	}
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && a != "-" {
			if valueTaking[a] && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

func nmapExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		positional := stripNmapFlags(params)
		if len(positional) == 0 {
			return OutputMsg("Nmap 7.94 ( https://nmap.org )\nUsage: nmap [Scan Type(s)] [Options] {target specification}")
		}
		target := positional[0]

		var match *NmapHost
		for i := range nmapHosts {
			if nmapHosts[i].IP == target {
				match = &nmapHosts[i]
				break
			}
		}

		header := fmt.Sprintf("Starting Nmap 7.94 ( https://nmap.org ) at %s EDT\nNmap scan report for %s\n",
			time.Now().Format("2006-01-02 15:04"), target)

		if match == nil {
			return OutputMsg(header +
				"Note: Host seems down. If it is really up, but blocking our ping probes, try -Pn\n" +
				"Nmap done: 1 IP address (0 hosts up) scanned in 0.32 seconds")
		}

		var b strings.Builder
		b.WriteString(header)
		b.WriteString("Host is up (0.0012s latency).\n")
		b.WriteString(fmt.Sprintf("Not shown: %d closed tcp ports (reset)\n", 1000-len(match.Ports)))
		b.WriteString(fmt.Sprintf("%-10s%-6s%-9s%s\n", "PORT", "STATE", "SERVICE", "VERSION"))
		for _, p := range match.Ports {
			b.WriteString(fmt.Sprintf("%-10s%-6s%-9s%s\n",
				fmt.Sprintf("%d/tcp", p.Port), "open", p.Service, p.Version))
		}
		b.WriteString("\nNmap done: 1 IP address (1 host up) scanned in 1.23 seconds")
		return OutputMsg(b.String())
	})
	return &cmd
}
```

- [ ] **Step 2: Add `time` import if missing**

Check the import block at the top of `internal/honeypot/filesystem/exec.go`. If `"time"` is not present, add it.

- [ ] **Step 3: Verify build**

Run: `go build ./internal/honeypot/filesystem/`
Expected: success.

- [ ] **Step 4: Stop for review**

---

## Task 9: Register `curl` and `nmap` nodes

**Files:**
- Modify: `internal/honeypot/filesystem/filesystem.go` (in `/usr/bin` children, after the existing `ping` node ~line 345)

- [ ] **Step 1: Insert nodes after the ping node**

In `internal/honeypot/filesystem/filesystem.go`, immediately after the closing `},` of the `ping` node (around line 345), insert:

```go
							{
								Name:      "curl",
								Path:      "/usr/bin/curl",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: curl [options] <url>\n Transfer a URL.",
								Exec:      curlExec,
							},
							{
								Name:      "nmap",
								Path:      "/usr/bin/nmap",
								Directory: false,
								Owner:     "root",
								Group:     "root",
								Mode:      0711,
								HelpText:  "Usage: nmap [Scan Type(s)] [Options] {target specification}",
								Exec:      nmapExec,
							},
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Stop for review**

---

## Task 10: Tests for curl and nmap

**Files:**
- Modify: `internal/honeypot/filesystem/exec_test.go` (append)

- [ ] **Step 1: Append tests**

Append to `internal/honeypot/filesystem/exec_test.go`:

```go
func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in       string
		wantNorm string
		wantHost string
	}{
		{"example.com", "http://example.com", "example.com"},
		{"http://Example.com/", "http://example.com", "example.com"},
		{"https://api.X.com/v1/", "https://api.x.com/v1", "api.x.com"},
		{"http://example.com/path?q=1", "http://example.com/path?q=1", "example.com"},
	}
	for _, c := range cases {
		gotN, gotH := normalizeURL(c.in)
		if gotN != c.wantNorm || gotH != c.wantHost {
			t.Errorf("normalizeURL(%q) = (%q,%q), want (%q,%q)", c.in, gotN, gotH, c.wantNorm, c.wantHost)
		}
	}
}

func TestStripCurlFlags(t *testing.T) {
	got := stripCurlFlags([]string{"-v", "-X", "POST", "-H", "X: y", "https://x.com", "-I"})
	want := []string{"https://x.com"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestCurlExecMissNoArgs(t *testing.T) {
	SetCurlResponses(nil)
	msgs := runExec(t, curlExec, nil)
	out := msgs[0].(OutputMsg)
	if !strings.Contains(string(out), "try 'curl --help'") {
		t.Errorf("got %q", string(out))
	}
}

func TestCurlExecMissUnknownHost(t *testing.T) {
	SetCurlResponses(nil)
	msgs := runExec(t, curlExec, []string{"https://nope.example/"})
	out := msgs[0].(OutputMsg)
	if !strings.Contains(string(out), "Could not resolve host: nope.example") {
		t.Errorf("got %q", string(out))
	}
}

func TestCurlExecHit(t *testing.T) {
	SetCurlResponses([]CurlResponse{{URL: "http://example.com/hello", Body: "hi!"}})
	defer SetCurlResponses(nil)
	msgs := runExec(t, curlExec, []string{"-v", "example.com/hello"})
	out := string(msgs[0].(OutputMsg))
	if !strings.Contains(out, "HTTP/1.1 200 OK") || !strings.Contains(out, "hi!") {
		t.Errorf("got %q", out)
	}
}

func TestCurlExecHeadOnly(t *testing.T) {
	SetCurlResponses([]CurlResponse{{URL: "http://example.com", Body: "body"}})
	defer SetCurlResponses(nil)
	msgs := runExec(t, curlExec, []string{"-I", "example.com"})
	out := string(msgs[0].(OutputMsg))
	if !strings.Contains(out, "HTTP/1.1 200 OK") || strings.Contains(out, "body") {
		t.Errorf("expected headers only, got %q", out)
	}
}

func TestStripNmapFlags(t *testing.T) {
	got := stripNmapFlags([]string{"-sV", "-p", "22,80", "-Pn", "10.0.0.1"})
	if len(got) != 1 || got[0] != "10.0.0.1" {
		t.Errorf("got %v", got)
	}
}

func TestNmapExecNoArgs(t *testing.T) {
	msgs := runExec(t, nmapExec, nil)
	out := string(msgs[0].(OutputMsg))
	if !strings.Contains(out, "Usage: nmap") {
		t.Errorf("got %q", out)
	}
}

func TestNmapExecMiss(t *testing.T) {
	SetNmapHosts(nil)
	msgs := runExec(t, nmapExec, []string{"10.99.99.99"})
	out := string(msgs[0].(OutputMsg))
	if !strings.Contains(out, "Host seems down") || !strings.Contains(out, "0 hosts up") {
		t.Errorf("got %q", out)
	}
}

func TestNmapExecHit(t *testing.T) {
	SetNmapHosts([]NmapHost{{
		IP: "10.0.0.5",
		Ports: []NmapPort{
			{Port: 22, Service: "ssh", Version: "OpenSSH 8.4"},
			{Port: 80, Service: "http", Version: "nginx 1.20"},
		},
	}})
	defer SetNmapHosts(nil)
	msgs := runExec(t, nmapExec, []string{"-sV", "10.0.0.5"})
	out := string(msgs[0].(OutputMsg))
	for _, want := range []string{"10.0.0.5", "22/tcp", "ssh", "OpenSSH 8.4", "80/tcp", "nginx 1.20", "1 host up"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/honeypot/filesystem/ -run "TestCurl|TestNmap|TestNormalize|TestStrip" -v`
Expected: PASS for all tests. (CGO/SQLite fallback: `go vet ./internal/honeypot/filesystem/`.)

- [ ] **Step 3: Stop for review**

---

## Task 11: Update `config.sample.json` with examples

**Files:**
- Modify: `config.sample.json`

- [ ] **Step 1: Inspect the existing sample file**

Run: `cat config.sample.json`
Note the existing top-level keys and trailing comma placement so the JSON stays valid.

- [ ] **Step 2: Add example entries**

Insert these two top-level keys (placement is wherever fits alphabetically; ensure commas are valid JSON):

```json
"curl_responses": [
    {"url": "http://internal.hardhat.local/status", "body": "{\"status\":\"ok\",\"uptime\":890123}"},
    {"url": "http://10.0.0.10/admin", "body": "<html><body><h1>HoneyBear Admin Console</h1></body></html>"}
],
"nmap_hosts": [
    {
        "ip": "10.0.0.10",
        "ports": [
            {"port": 22, "service": "ssh", "version": "OpenSSH 8.4"},
            {"port": 80, "service": "http", "version": "nginx 1.20.1"},
            {"port": 443, "service": "https", "version": "nginx 1.20.1"}
        ]
    }
]
```

- [ ] **Step 3: Validate JSON**

Run: `python3 -m json.tool config.sample.json > /dev/null`
Expected: no output, exit 0.

- [ ] **Step 4: Stop for review**

---

## Task 12: Full build + final smoke

**Files:** none (verification only)

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 2: Full test (if CGO available)**

Run: `go test ./...`
Expected: all packages PASS. If CGO unavailable for SQLite/Fyne packages, at minimum the filesystem package tests must pass: `go test ./internal/honeypot/filesystem/ -v`.

- [ ] **Step 3: Format**

Run: `go fmt ./...`
Expected: no output (or list of reformatted files — that's fine).

- [ ] **Step 4: Hand back to maintainer for manual SSH smoke + commit**

Suggest the maintainer test interactively:
- `ssh -p 1337 anyuser@localhost` then:
  - `cat /home/you/patch.md` (should dump contents inline, no viewport).
  - `vi /home/you/patch.md` (should open viewport with `[readonly]` status line at the bottom).
  - `less /home/you/patch.md` / `more /home/you/patch.md` (same viewport as vi).
  - `curl example.com` (should error with "Could not resolve host"); with a configured URL in `~/Library/Application Support/HoneyBearHoneyPot/...` config, should return body.
  - `nmap 10.0.0.10` (configured) → port table; `nmap 1.2.3.4` (not configured) → "0 hosts up".

---

## Self-review notes

- All spec requirements have a task: config types (Task 1-2), cat split (3-5), tests (6, 10), curl (7), nmap (8), node registration (5, 9), sample config (11).
- No placeholders; every step has the exact code or command.
- Type/method name consistency checked: `CurlResponse`/`NmapHost`/`NmapPort`, `SetCurlResponses`/`SetNmapHosts`, `curlExec`/`nmapExec`/`viExec`/`catExec`, `normalizeURL`/`stripCurlFlags`/`stripNmapFlags`/`hasFlag`/`viStatusLine`.
- `help` command's `SetRunningCmd("cat")` is updated to `"vi"` in Task 3 step 4 — without this, `help` would break after the model.go rename.
