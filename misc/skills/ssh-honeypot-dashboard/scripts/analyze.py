#!/usr/bin/env python3
"""
Analyze SSH honeypot JSON events and output metrics as JSON to stdout.

Usage: python3 analyze.py <input.json> [output_metrics.json]

Input: JSON array of event objects with fields:
  id, user, host, app, source, type, action, timestamp

Output: JSON object with all computed metrics for dashboard rendering.
"""

import json
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime


def extract_ip(host: str) -> str:
    m = re.match(r"\[([^\]]+)\]", host)
    if m:
        return m.group(1)
    return host.rsplit(":", 1)[0]


def trunc(s: str, n: int = 80) -> str:
    return s[:n] + "..." if len(s) > n else s


RECON_PATTERNS = [
    "uname", "whoami", "id", "hostname", "uptime", "nproc", "lscpu",
    "grep.*cpuinfo", "env", "history", "pwd", "ssh -V", "uname -m",
    "cat /etc", "ifconfig", "ip addr",
]
FILESYSTEM_PATTERNS = [
    "ls", "cd ", "cat ", "find ", "locate", "mkdir", "rm ", "cp ", "mv ",
    "chmod", "chown", "mount", "df ", "du ",
]
NETWORK_PATTERNS = [
    "netstat", "ss ", "curl", "wget", "ping", "nmap", "nc ", "telnet", "iptables",
]
PROCESS_PATTERNS = ["ps ", "top", "kill", "pkill", "service", "systemctl"]
MALWARE_PATTERNS = ["wget.*http", "curl.*http", "tftp", "busybox", "chmod +x", r"\./"]
PERSIST_PATTERNS = ["crontab", "useradd", "passwd", "authorized_keys"]
MISC_KEYWORDS = ["exit", "echo", "help", "ctf", "celebrate", "matrix"]

CATEGORIES = {
    "Recon": RECON_PATTERNS,
    "File System": FILESYSTEM_PATTERNS,
    "Network": NETWORK_PATTERNS,
    "Process": PROCESS_PATTERNS,
    "Malware/DL": MALWARE_PATTERNS,
    "Persistence": PERSIST_PATTERNS,
}

DOWNLOAD_MARKERS = ["wget", "curl", "tftp", "busybox", "chmod +x", "./"]


def categorize(action: str) -> str:
    al = action.lower()
    for cat, patterns in CATEGORIES.items():
        for p in patterns:
            if re.search(p, al):
                return cat
    if any(x in al for x in MISC_KEYWORDS):
        return "Misc/Exit"
    return "Other"


def analyze(data: list[dict]) -> dict:
    total_events = len(data)
    typed_events = [e for e in data if e["type"] == "typed"]
    login_events = [e for e in data if e["type"] == "login"]

    users = set(e["user"] for e in data)
    ips = set(extract_ip(e["host"]) for e in data)

    timestamps_sorted = sorted(e["timestamp"] for e in data)
    date_start = timestamps_sorted[0][:10]
    date_end = timestamps_sorted[-1][:10]

    # --- Daily breakdown by type ---
    daily_login = Counter()
    daily_typed = Counter()
    for e in data:
        d = e["timestamp"][:10]
        if e["type"] == "login":
            daily_login[d] += 1
        elif e["type"] == "typed":
            daily_typed[d] += 1
    all_days = sorted(set(list(daily_login.keys()) + list(daily_typed.keys())))

    # --- Hourly ---
    hourly = Counter(int(e["timestamp"][11:13]) for e in data)

    # --- Day of week ---
    dow_names = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
    dow_counts = Counter()
    for e in data:
        ts = datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
        dow_counts[ts.weekday()] += 1

    # --- Monthly ---
    monthly = Counter(e["timestamp"][:7] for e in data)

    # --- Top users ---
    user_counts = Counter(e["user"] for e in data)

    # --- Top IPs ---
    ip_counts = Counter(extract_ip(e["host"]) for e in data)

    # --- Command categories ---
    cat_counts = Counter(categorize(e["action"]) for e in typed_events)

    # --- Top commands ---
    action_counts = Counter(e["action"] for e in typed_events)

    # --- Event type distribution ---
    type_counts = Counter(e["type"] for e in data)

    # --- Login usernames ---
    login_user_counts = Counter(e["user"] for e in login_events)

    # --- Sessions ---
    events_sorted = sorted(data, key=lambda e: (e["user"], e["host"], e["timestamp"]))
    sessions = []
    current_session = None
    for e in events_sorted:
        ts = datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
        key = (e["user"], e["host"])
        if current_session and current_session["key"] == key:
            gap = (ts - current_session["end"]).total_seconds()
            if gap < 3600:
                current_session["end"] = ts
                current_session["events"] += 1
                continue
        if current_session:
            sessions.append(current_session)
        current_session = {"key": key, "start": ts, "end": ts, "events": 1, "user": e["user"]}
    if current_session:
        sessions.append(current_session)

    total_sessions = len(sessions)
    session_durations = [(s["end"] - s["start"]).total_seconds() for s in sessions]
    avg_session_dur = sum(session_durations) / len(session_durations) if session_durations else 0

    # --- Malware attempts ---
    download_cmds = [
        e for e in typed_events
        if any(p in e["action"].lower() for p in DOWNLOAD_MARKERS)
    ]
    malware_actions = Counter()
    malware_ips = defaultdict(set)
    for e in download_cmds:
        a = e["action"]
        if len(a) > 20:
            malware_actions[a] += 1
            malware_ips[a].add(extract_ip(e["host"]))

    malware_table = []
    for a, c in sorted(malware_actions.items(), key=lambda x: -x[1])[:15]:
        malware_table.append({
            "command_short": trunc(a, 80),
            "command_full": a,
            "count": c,
            "ips": list(malware_ips[a])[:3],
        })

    # --- Heatmap (dow x hour) ---
    heatmap = defaultdict(int)
    for e in data:
        ts = datetime.fromisoformat(e["timestamp"].replace("Z", "+00:00"))
        heatmap[(ts.weekday(), ts.hour)] += 1
    heatmap_data = [
        {"x": h, "y": d, "v": heatmap.get((d, h), 0)}
        for d in range(7) for h in range(24)
    ]

    # --- Insights ---
    top_user = user_counts.most_common(1)[0] if user_counts else ("", 0)
    top_ip = ip_counts.most_common(1)[0] if ip_counts else ("", 0)
    top_cmd = action_counts.most_common(1)[0] if action_counts else ("", 0)
    root_login_pct = (
        login_user_counts.get("root", 0) / len(login_events) * 100
        if login_events else 0
    )

    # Find peak hours
    peak_hours = sorted(hourly.items(), key=lambda x: -x[1])[:5]
    peak_range_start = min(h for h, _ in peak_hours)
    peak_range_end = max(h for h, _ in peak_hours) + 1

    # Busiest day
    daily_total = Counter()
    for e in data:
        daily_total[e["timestamp"][:10]] += 1
    busiest_day = daily_total.most_common(1)[0] if daily_total else ("", 0)

    insights = [
        f"{root_login_pct:.1f}% of login attempts use \"root\" — classic brute-force pattern"
        if root_login_pct > 50 else None,
        f"Top attacker IP {top_ip[0]} accounts for {top_ip[1]/total_events*100:.0f}% of all events ({top_ip[1]:,} events)",
        f"Most active hours are {peak_range_start:02d}:00-{peak_range_end:02d}:00 UTC",
        f"Busiest day was {busiest_day[0]} with {busiest_day[1]:,} events",
        f"{len(download_cmds)} malware download/execution attempts detected" if download_cmds else None,
        f"Dominant bot fingerprint: \"{trunc(top_cmd[0], 40)}\" appearing {top_cmd[1]:,} times"
        if top_cmd[1] > 100 else None,
    ]
    insights = [i for i in insights if i]

    return {
        "total_events": total_events,
        "unique_ips": len(ips),
        "unique_users": len(users),
        "login_count": len(login_events),
        "typed_count": len(typed_events),
        "total_sessions": total_sessions,
        "avg_session_dur": round(avg_session_dur),
        "malware_attempts": len(download_cmds),
        "date_start": date_start,
        "date_end": date_end,
        "insights": insights,
        "daily_labels": all_days,
        "daily_login": [daily_login.get(d, 0) for d in all_days],
        "daily_typed": [daily_typed.get(d, 0) for d in all_days],
        "monthly_labels": [m for m in sorted(monthly.keys())],
        "monthly_values": [monthly[m] for m in sorted(monthly.keys())],
        "hourly_labels": [f"{h:02d}:00" for h in range(24)],
        "hourly_values": [hourly.get(h, 0) for h in range(24)],
        "dow_labels": dow_names,
        "dow_values": [dow_counts.get(i, 0) for i in range(7)],
        "top_users": {"labels": [u for u, _ in user_counts.most_common(15)],
                      "values": [c for _, c in user_counts.most_common(15)]},
        "top_ips": {"labels": [i for i, _ in ip_counts.most_common(15)],
                    "values": [c for _, c in ip_counts.most_common(15)]},
        "categories": {"labels": [c for c, _ in cat_counts.most_common()],
                       "values": [n for _, n in cat_counts.most_common()]},
        "top_commands": {"labels": [trunc(a, 45) for a, _ in action_counts.most_common(12)],
                         "values": [c for _, c in action_counts.most_common(12)]},
        "event_types": {"labels": list(type_counts.keys()),
                        "values": list(type_counts.values())},
        "login_users": {"labels": [u for u, _ in login_user_counts.most_common(10)],
                        "values": [c for _, c in login_user_counts.most_common(10)]},
        "malware_table": malware_table,
        "heatmap": heatmap_data,
    }


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <input.json> [output_metrics.json]", file=sys.stderr)
        sys.exit(1)

    input_path = sys.argv[1]
    output_path = sys.argv[2] if len(sys.argv) > 2 else None

    with open(input_path, "r") as f:
        data = json.load(f)

    metrics = analyze(data)
    result = json.dumps(metrics)

    if output_path:
        with open(output_path, "w") as f:
            f.write(result)
        print(f"Metrics written to {output_path}", file=sys.stderr)
    else:
        print(result)


if __name__ == "__main__":
    main()
