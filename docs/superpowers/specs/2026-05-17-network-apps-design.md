# Network Apps & Editor Polish — Design

Date: 2026-05-17
Branch: `network-apps`

## Goal

Add two network-flavored honeypot commands (`curl`, `nmap`) and split the current `cat` into two commands: a real-cat-style streaming dump (`cat`) and a vi-flavored paged viewer (`vi`). `less` and `more` keep the existing paged-viewport behavior.

## Scope

Four changes inside `internal/honeypot/filesystem/` plus a small rename in `internal/honeypot/model.go` and two new config fields.

1. **`vi`** — new node. Takes over today's `cat` UX (paged viewport via `FileContentsMsg` + `SetRunningCmd`), with a vi-flavored status bar appended to the rendered buffer.
2. **`cat`** — replaced. Streams full file contents via `OutputMsg`; no viewport, no `SetRunningCmd`.
3. **`curl`** — new. Config-driven URL→body map. On miss, prints the standard `Could not resolve host` error.
4. **`nmap`** — new. Config-driven IP→[{port, service, version}] map. On miss, prints a scan that finds nothing.

`less` and `more` keep today's paged viewport behavior by sharing the new `viExec`.

## Config additions

Extend `internal/config/Config`:

```go
type CurlResponse struct {
    URL  string `json:"url"`   // exact match, normalized
    Body string `json:"body"`
}

type NmapPort struct {
    Port    int    `json:"port"`
    Service string `json:"service"`
    Version string `json:"version,omitempty"`
}

type NmapHost struct {
    IP    string     `json:"ip"`
    Ports []NmapPort `json:"ports"`
}

type Config struct {
    // ... existing fields ...
    CurlResponses []CurlResponse `json:"curl_responses,omitempty"`
    NmapHosts     []NmapHost     `json:"nmap_hosts,omitempty"`
}
```

`merge()` gets two new clauses (replace dst slice if `len(src.X) > 0`). `config.sample.json` gets example entries for both.

URL normalization for curl lookup: prepend `http://` if no scheme, lowercase scheme + host, trim trailing slash on path. Lookup against `config.Active.CurlResponses` is exact match against the same normalization.

## Command behaviors

### `cat` (new — replaces current `cat` exec)

- `cat <path>`: resolve file via existing `GetNodeByPath`, call `target.Open()`, emit `OutputMsg(string(data))`. No `SetRunningCmd`, no viewport.
- `cat` (no args): return immediately with no output. (Real cat reads stdin; hanging would be worse than no-op.)
- `cat <dir>`: `cat: <name>: Is a directory`.
- Missing file: `cat: <path>: No such file or directory`.

### `vi` (new — uses today's catExec logic)

- Sets `SetRunningCmd("vi")`, returns `FileContentsMsg`.
- `model.go`: rename the `"cat"` case in `Update` (line 264) and `View` (line 298) to `"vi"`. Footer hint becomes `:q or ctrl + c to exit`.
- Status bar: `viExec` appends a final line to the file contents before returning `FileContentsMsg`:
  ```
  "filename" [readonly]  N lines, M bytes
  ```
- No-arg `vi`: shows a fake splash buffer (`VIM - Vi IMproved`, `type :q to exit`). User still exits with ctrl+c since we don't parse keystrokes — same as today's cat behavior.
- `less` and `more` continue to point at the same exec (`viExec`); they get the same viewport experience as `vi`.

### `curl <url> [flags]`

- Strip flags: drop tokens starting with `-`. For known value-taking flags (`-X`, `-H`, `-d`, `-o`, `-A`, `-e`, `-u`), also drop the following token. First non-flag arg is the URL.
- Normalize URL and look up in `config.Active.CurlResponses`.
- Hit: print a minimal fake header block + blank line + body:
  ```
  HTTP/1.1 200 OK
  Content-Type: text/html
  Content-Length: <len>

  <body>
  ```
  If `-I` is present, print only the header block.
- Miss: `curl: (6) Could not resolve host: <host>`.
- No args: `curl: try 'curl --help' or 'curl --manual' for more information`.

### `nmap [flags] <target>`

- Strip flags (same rule as curl: drop `-` tokens; value-taking flags `-p`, `-oN`, `-oX`, `-oG`, `-iL`, `-e`, `-S` also drop the next token). Targets are positional.
- Single literal target only (v1). No CIDR parsing — `10.0.0.0/24` is looked up literally and yields "0 hosts up".
- Output (single `OutputMsg`; no real delay, since blocking would freeze the interactive prompt):
  ```
  Starting Nmap 7.94 ( https://nmap.org ) at <date> EDT
  Nmap scan report for <target>
  Host is up (0.0012s latency).
  Not shown: 996 closed tcp ports (reset)
  PORT      STATE SERVICE  VERSION
  22/tcp    open  ssh      OpenSSH 8.4
  ...

  Nmap done: 1 IP address (1 host up) scanned in 1.23 seconds
  ```
- Miss: skip the port table; end with
  ```
  Note: Host seems down. If it is really up, but blocking our ping probes, try -Pn
  Nmap done: 1 IP address (0 hosts up) scanned in 0.32 seconds
  ```
- No args:
  ```
  Nmap 7.94 ( https://nmap.org )
  Usage: nmap [Scan Type(s)] [Options] {target specification}
  ```

## Node registration & logging

- Add `curl` and `nmap` nodes in `filesystem.go` under `/usr/bin` (alongside `ping` at line 330). Both `Fun: false`, mode 0711, owner root, with `HelpText` matching real man-page summaries.
- `cat` node's `Exec` swaps from current `catExec` → new streaming `catExec`. `vi`, `less`, `more` use `viExec` (the renamed paged version of today's catExec, with status-line addition).
- Logging: existing event flow in `pot.go`/`exec.go` already captures every typed command. No new logging code needed.

## Files touched

- `internal/config/config.go` — new fields + merge clauses.
- `internal/honeypot/filesystem/exec.go` — new `curlExec`, `nmapExec`, `viExec`; rewrite `catExec`.
- `internal/honeypot/filesystem/filesystem.go` — register `curl`, `nmap`, `vi` nodes; point `cat`/`less`/`more` at the correct execs.
- `internal/honeypot/model.go` — rename `"cat"` → `"vi"` in the two switch cases (Update line 264, View line 298), update footer hint.
- `config.sample.json` — example `curl_responses` and `nmap_hosts` entries.

## Testing

Unit tests in `internal/honeypot/filesystem/` for each new/changed exec:

- `cat`: hit, missing file, directory, no args.
- `vi`: status line is appended; correct line/byte counts; no-arg splash.
- `curl`: hit, miss (error format), `-I` headers-only, no args, flag stripping with value-taking flags.
- `nmap`: hit (port table format), miss ("host seems down"), no args, flag stripping.

Patterns follow existing tests in `dirs_test.go` and `autocomplete_test.go`.

## Out of scope (v1)

- CIDR / range targets for nmap.
- Real network delays / progressive output streaming for nmap.
- curl POST body display, redirect following, cookie jars.
- vi modal keystroke parsing (`:q`, insert-mode rejection).
