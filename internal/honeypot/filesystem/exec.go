package filesystem

import (
	"fmt"
	"math/rand/v2"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func echoExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		interpret := false
		noNewline := false
		args := []string{}

		for _, p := range params {
			if strings.HasPrefix(p, "-") && len(p) > 1 {
				for i := 1; i < len(p); i++ {
					switch p[i] {
					case 'e':
						interpret = true
					case 'E':
						interpret = false
					case 'n':
						noNewline = true
					}
				}
			} else {
				args = append(args, p)
			}
		}

		output := strings.Join(args, " ")
		if interpret {
			output = interpretEscapes(output)
		}

		if noNewline {
			// Currently OutputMsg in model.go adds newlines,
			// so -n might not be fully effective without model.go changes.
			// But we'll implement the logic here.
			return OutputMsg(output)
		}

		return OutputMsg(output)
	})
	return &cmd
}

func interpretEscapes(s string) string {
	var res strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'a':
				res.WriteByte('\a')
			case 'b':
				res.WriteByte('\b')
			case 'c':
				return res.String()
			case 'e', 'E':
				res.WriteByte('\x1b')
			case 'f':
				res.WriteByte('\f')
			case 'n':
				res.WriteByte('\n')
			case 'r':
				res.WriteByte('\r')
			case 't':
				res.WriteByte('\t')
			case 'v':
				res.WriteByte('\v')
			case '\\':
				res.WriteByte('\\')
			case 'x':
				if i+2 < len(s) {
					hex := s[i+1 : i+3]
					if val, err := strconv.ParseUint(hex, 16, 8); err == nil {
						res.WriteByte(byte(val))
						i += 2
					} else {
						res.WriteString("\\x")
					}
				} else {
					res.WriteString("\\x")
				}
			case '0':
				// Octal: \0NNN (up to 3 digits)
				if i+1 < len(s) {
					end := i + 4
					if end > len(s) {
						end = len(s)
					}
					octalStr := ""
					for j := i + 1; j < end; j++ {
						if s[j] >= '0' && s[j] <= '7' {
							octalStr += string(s[j])
						} else {
							break
						}
					}
					if len(octalStr) > 0 {
						if val, err := strconv.ParseUint(octalStr, 8, 8); err == nil {
							res.WriteByte(byte(val))
							i += len(octalStr)
						} else {
							res.WriteByte('0')
						}
					} else {
						res.WriteByte('0')
					}
				} else {
					res.WriteByte('0')
				}
			default:
				res.WriteByte('\\')
				res.WriteByte(s[i])
			}
		} else if s[i] == 'x' && i+2 < len(s) {
			// Lenient hex handling for cases where the backslash was stripped by a local shell
			hex := s[i+1 : i+3]
			if val, err := strconv.ParseUint(hex, 16, 8); err == nil {
				res.WriteByte(byte(val))
				i += 2
			} else {
				res.WriteByte('x')
			}
		} else {
			res.WriteByte(s[i])
		}
	}
	return res.String()
}

func bearSayExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmds := []tea.Cmd{}

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		output := `
				  __         __
				 /  \.-"""-./  \
				\    -   -    /
				 |   o   o   |
				 \  .-'''-.  /
				  '-\__Y__/-'
				     '---'
				      (\__/)
				      (='.'=)
				      (")_(")

				%s

			`

		if len(params) == 0 {
			defaults := []string{
				"Hello, world!",
				"You're in!",
				"I don't play well with others",
				"Hack the planet!",
				"Its in the place that I put that thing that time.",
				"Stay curious!",
			}

			output = fmt.Sprintf(output, defaults[rand.IntN(len(defaults))])
		} else {
			output = fmt.Sprintf(output, strings.Join(params, " "))
		}

		return OutputMsg(output)
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

func neofetchExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmds := []tea.Cmd{}

	cmds = append(cmds, tea.Cmd(func() tea.Msg {
		logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

		logo := `
       __         __
      /  \.-"""-./  \
     \    -   -    /
      |   o   o   |
      \  .-'''-.  /
       '-\__Y__/-'
          '---'
`
		logoLines := strings.Split(logo, "\n")

		info := []struct {
			key string
			val string
		}{
			{"OS", "Hardhat Linux"},
			{"Host", "HoneyBear (18.2)"},
			{"Kernel", "6.22.0-81-generic"},
			{"Uptime", "10 days, 23 hours, 21 mins"},
			{"Packages", "1337 (dpkg)"},
			{"Shell", "bearshell"},
			{"Resolution", "1920x1080"},
			{"DE", "HoneyDesktop"},
			{"WM", "HoneyWindow"},
			{"Theme", "HoneyDark [GTK2/3]"},
			{"Icons", "HoneyIcons [GTK2/3]"},
			{"Terminal", "/dev/pts/0"},
			{"CPU", "HoneyBear CPU @ 4.0GHz"},
			{"GPU", "HoneyBear GPU"},
			{"Memory", "1337MiB / 2048MiB"},
		}

		var outputLines []string
		maxLines := len(logoLines)
		if len(info) > maxLines {
			maxLines = len(info)
		}

		// User@Host line
		userHost := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("root@honeybear")
		separator := "----------------"

		// Prepend empty lines to logo if it's shorter than info to center it roughly or top align
		// Actually neofetch usually top aligns.

		for i := 0; i < maxLines; i++ {
			logoLine := ""
			if i < len(logoLines) {
				logoLine = logoStyle.Render(logoLines[i])
			}

			// Pad logo line to a fixed width
			logoWidth := 30
			currentLen := 0
			if i < len(logoLines) {
				currentLen = lipgloss.Width(logoLines[i]) // Use raw string for length
			}
			if logoWidth < currentLen {
				logoWidth = currentLen
			}
			padding := strings.Repeat(" ", logoWidth-currentLen)
			logoPart := logoLine + padding

			infoPart := ""
			if i == 0 {
				infoPart = userHost
			} else if i == 1 {
				infoPart = valStyle.Render(separator)
			} else if i-2 < len(info) && i >= 2 {
				idx := i - 2
				infoPart = keyStyle.Render(info[idx].key+":") + " " + valStyle.Render(info[idx].val)
			}

			outputLines = append(outputLines, logoPart+infoPart)
		}

		// Color blocks at the bottom
		blocks := ""
		for i := 0; i < 8; i++ {
			blocks += lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i))).Render("   ")
		}
		blocks += "\n"
		for i := 8; i < 16; i++ {
			blocks += lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", i))).Render("   ")
		}

		outputLines = append(outputLines, "", blocks)

		return OutputMsg("\n" + strings.Join(outputLines, "\n"))
	}))

	batch := tea.Batch(cmds...)
	return &batch
}

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

func idExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		if user == "root" {
			return OutputMsg("uid=0(root) gid=0(root) groups=0(root)")
		}
		return OutputMsg("uid=1000(you) gid=1000(you) groups=1000(you),27(sudo)")
	})
	return &cmd
}

func psExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		output := fmt.Sprintf("%-8s %-5s %-5s %-5s %-8s %-8s %-5s %s\n", "USER", "PID", "%CPU", "%MEM", "VSZ", "RSS", "TTY", "COMMAND")
		processes := []struct {
			user, pid, cpu, mem, vsz, rss, tty, cmd string
		}{
			{"root", "1", "0.0", "0.1", "168244", "12544", "?", "/sbin/init"},
			{"root", "2", "0.0", "0.0", "0", "0", "?", "[kthreadd]"},
			{"root", "3", "0.0", "0.0", "0", "0", "?", "[rcu_gp]"},
			{"you", "452", "0.1", "0.5", "23452", "8432", "pts/0", "-bash"},
			{"you", "1337", "4.2", "1.2", "104320", "24512", "pts/0", "./honeybear --no-gui"},
			{"root", "2048", "0.0", "0.2", "72432", "4124", "?", "/usr/sbin/sshd -D"},
		}

		for _, p := range processes {
			output += fmt.Sprintf("%-8s %-5s %-5s %-5s %-8s %-8s %-5s %s\n", p.user, p.pid, p.cpu, p.mem, p.vsz, p.rss, p.tty, p.cmd)
		}
		return OutputMsg(output)
	})
	return &cmd
}

func envExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		envVars := []string{}
		for k, v := range env {
			envVars = append(envVars, k+"="+v)
		}
		sort.Strings(envVars)
		return OutputMsg(strings.Join(envVars, "\n"))
	})
	return &cmd
}

func exportExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	if len(params) == 0 {
		cmd := tea.Cmd(func() tea.Msg {
			envVars := []string{}
			for k, v := range env {
				envVars = append(envVars, "export "+k+"=\""+v+"\"")
			}
			sort.Strings(envVars)
			return OutputMsg(strings.Join(envVars, "\n"))
		})
		return &cmd
	}

	cmds := []tea.Cmd{}
	for _, p := range params {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) == 2 {
			key, val := parts[0], parts[1]
			cmds = append(cmds, func() tea.Msg {
				return SetEnvMsg{Key: key, Value: val}
			})
		}
	}

	batch := tea.Batch(cmds...)
	return &batch
}

func unsetExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmds := []tea.Cmd{}
	for _, p := range params {
		key := p
		cmds = append(cmds, func() tea.Msg {
			return UnsetEnvMsg(key)
		})
	}
	batch := tea.Batch(cmds...)
	return &batch
}

func netstatExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		output := "Active Internet connections (only servers)\n"
		output += fmt.Sprintf("%-5s %-6s %-6s %-20s %-20s %-10s\n", "Proto", "Recv-Q", "Send-Q", "Local Address", "Foreign Address", "State")
		ports := []struct {
			proto, recv, send, local, foreign, state string
		}{
			{"tcp", "0", "0", "0.0.0.0:22", "0.0.0.0:*", "LISTEN"},
			{"tcp", "0", "0", "0.0.0.0:1337", "0.0.0.0:*", "LISTEN"},
			{"tcp", "0", "0", "127.0.0.1:3306", "0.0.0.0:*", "LISTEN"},
			{"tcp6", "0", "0", ":::80", ":::*", "LISTEN"},
		}

		for _, p := range ports {
			output += fmt.Sprintf("%-5s %-6s %-6s %-20s %-20s %-10s\n", p.proto, p.recv, p.send, p.local, p.foreign, p.state)
		}
		return OutputMsg(output)
	})
	return &cmd
}

func whoamiExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		return OutputMsg(user)
	})
	return &cmd
}

func sudoExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	// If no arguments, or sudo -h etc not supported for now, return simple help or prompt
	if len(params) == 0 {
		cmd := tea.Cmd(func() tea.Msg { return OutputMsg("usage: sudo [command]") })
		return &cmd
	}

	// params[0] is the command to run as root
	// params[1:] are the args
	newCmd, err := RunNode(dir, params[0], params[1:], "root", "root", env)
	if err != nil {
		cmd := tea.Cmd(func() tea.Msg { return OutputMsg(err.Error()) })
		return &cmd
	}
	return newCmd
}

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

		body, found := curlResponseBodies[normalized]
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

// expandNmapTarget parses one target spec into the list of IPs to scan.
// Supports a bare IP/host, CIDR (10.0.0.0/24), and last-octet hyphen
// ranges (10.0.0.1-50). CIDR larger than /16 falls back to a single-IP
// scan against the literal string so an overly broad scan still produces
// a sensible (and quick) miss.
func expandNmapTarget(target string) []string {
	if strings.Contains(target, "/") {
		_, ipnet, err := net.ParseCIDR(target)
		if err != nil {
			return []string{target}
		}
		ones, bits := ipnet.Mask.Size()
		if bits-ones > 16 {
			return []string{target}
		}
		var ips []string
		for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); ip = nextIP(ip) {
			ips = append(ips, ip.String())
		}
		return ips
	}
	if idx := strings.LastIndex(target, "-"); idx > 0 {
		base := target[:idx]
		endStr := target[idx+1:]
		if lastDot := strings.LastIndex(base, "."); lastDot > 0 {
			prefix := base[:lastDot+1]
			startStr := base[lastDot+1:]
			start, err1 := strconv.Atoi(startStr)
			end, err2 := strconv.Atoi(endStr)
			if err1 == nil && err2 == nil && start >= 0 && end <= 255 && start <= end {
				ips := make([]string, 0, end-start+1)
				for i := start; i <= end; i++ {
					ips = append(ips, prefix+strconv.Itoa(i))
				}
				return ips
			}
		}
	}
	return []string{target}
}

func nextIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			break
		}
	}
	return out
}

func nmapExec(dir *Node, params []string, user, group string, env map[string]string) *tea.Cmd {
	cmd := tea.Cmd(func() tea.Msg {
		positional := stripNmapFlags(params)
		if len(positional) == 0 {
			return OutputMsg("Nmap 7.94 ( https://nmap.org )\nUsage: nmap [Scan Type(s)] [Options] {target specification}")
		}
		ips := expandNmapTarget(positional[0])

		var b strings.Builder
		b.WriteString(fmt.Sprintf("Starting Nmap 7.94 ( https://nmap.org ) at %s EDT\n",
			time.Now().Format("2006-01-02 15:04")))

		hostsUp := 0
		for _, ip := range ips {
			match, ok := nmapHostsByIP[ip]
			if !ok {
				continue
			}
			hostsUp++
			b.WriteString(fmt.Sprintf("Nmap scan report for %s\n", ip))
			b.WriteString("Host is up (0.0012s latency).\n")
			b.WriteString(fmt.Sprintf("Not shown: %d closed tcp ports (reset)\n", 1000-len(match.Ports)))
			b.WriteString(fmt.Sprintf("%-10s%-6s%-9s%s\n", "PORT", "STATE", "SERVICE", "VERSION"))
			for _, p := range match.Ports {
				b.WriteString(fmt.Sprintf("%-10s%-6s%-9s%s\n",
					fmt.Sprintf("%d/tcp", p.Port), "open", p.Service, p.Version))
			}
			b.WriteString("\n")
		}

		if len(ips) == 1 && hostsUp == 0 {
			b.WriteString(fmt.Sprintf("Nmap scan report for %s\n", ips[0]))
			b.WriteString("Note: Host seems down. If it is really up, but blocking our ping probes, try -Pn\n")
		}

		ipNoun := "IP address"
		if len(ips) != 1 {
			ipNoun = "IP addresses"
		}
		hostNoun := "host"
		if hostsUp != 1 {
			hostNoun = "hosts"
		}
		b.WriteString(fmt.Sprintf("Nmap done: %d %s (%d %s up) scanned in %.2f seconds",
			len(ips), ipNoun, hostsUp, hostNoun, 0.32+float64(len(ips))*0.01))

		return OutputMsg(b.String())
	})
	return &cmd
}
