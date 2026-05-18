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
