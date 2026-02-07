---
name: ssh-honeypot-dashboard
description: >
  Analyze SSH honeypot event logs and generate an interactive HTML dashboard
  with charts and security insights. Use when the user provides a JSON file
  of SSH honeypot events and wants a visual analysis, dashboard, or report.
  Triggers: SSH honeypot, honeypot analysis, honeypot dashboard, honeypot events,
  SSH attack analysis, SSH log analysis, intrusion analysis, brute-force analysis.
  Input format: JSON array of objects with fields: id, user, host, app, source,
  type (login/typed/taskCompleted), action, timestamp.
---

# SSH Honeypot Dashboard

Generate an interactive dark-themed HTML dashboard from SSH honeypot JSON event logs.

## Workflow

1. Locate the user's honeypot JSON file
2. Run `scripts/analyze.py <input.json> <metrics.json>` to extract metrics
3. Run `scripts/render_dashboard.py <metrics.json> <output.html>` to render the dashboard
4. Deliver the HTML file to the user

## Input Format

JSON array where each element has:

```json
{
  "id": 1,
  "user": "root",
  "host": "192.168.1.1:54321",
  "app": "ssh",
  "source": "user",
  "type": "login",
  "action": "Logged in!",
  "timestamp": "2025-01-15T08:25:44Z"
}
```

`type` is one of: `login`, `typed`, `taskCompleted`.

## Dashboard Contents

The generated dashboard includes:

- **Stat cards**: total events, unique IPs, unique users, logins, commands, sessions, avg session duration, malware attempts
- **Auto-generated insights**: root login %, top attacker IP, peak hours, busiest day, malware count, dominant bot fingerprint
- **10 Chart.js charts**: daily timeline (stacked login/typed), monthly trend, hourly activity, top usernames, top IPs, command categories, event types, top commands, day-of-week, login usernames
- **Heatmap**: day-of-week x hour-of-day activity grid
- **Malware table**: top download/execution attempts with click-to-expand modal showing full command and copy button

## Scripts

### `scripts/analyze.py`

```
python3 scripts/analyze.py <input.json> [output_metrics.json]
```

Reads raw events, computes all metrics, outputs JSON. If no output path given, prints to stdout.

### `scripts/render_dashboard.py`

```
python3 scripts/render_dashboard.py <metrics.json> <output.html>
```

Takes metrics JSON from analyze.py and produces a self-contained HTML dashboard. Requires no dependencies beyond Python stdlib. The HTML loads Chart.js 4.4.1 from CDN.

## Customization

If the user requests changes to the dashboard (colors, additional charts, different layout), edit `scripts/render_dashboard.py` directly. The render function returns a single HTML string with all CSS/JS inline.
