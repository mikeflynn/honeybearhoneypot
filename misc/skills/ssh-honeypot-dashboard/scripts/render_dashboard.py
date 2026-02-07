#!/usr/bin/env python3
"""
Render an interactive HTML dashboard from honeypot metrics JSON.

Usage: python3 render_dashboard.py <metrics.json> <output.html>

Input: metrics JSON produced by analyze.py
Output: Self-contained HTML dashboard with Chart.js
"""

import html
import json
import sys


def render(metrics: dict) -> str:
    m = metrics

    # Build malware table rows
    malware_rows = ""
    for entry in m.get("malware_table", []):
        escaped_short = html.escape(entry["command_short"])
        escaped_full = html.escape(entry["command_full"])
        ips_str = ", ".join(entry["ips"])
        malware_rows += (
            f'<tr class="malware-row" data-full="{escaped_full}">'
            f'<td class="cmd-cell"><span class="cmd-short">{escaped_short}</span>'
            f'<span class="click-hint">click to expand</span></td>'
            f'<td>{entry["count"]}</td><td>{ips_str}</td></tr>\n'
        )

    # Build insights list
    insights_html = ""
    for insight in m.get("insights", []):
        insights_html += f"<li>{html.escape(insight)}</li>\n"

    # JSON-encode chart data
    def j(v):
        return json.dumps(v)

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SSH Honeypot Dashboard</title>
<script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.4.1/chart.umd.min.js"></script>
<style>
  :root {{
    --bg: #0f1117; --card: #1a1d27; --border: #2a2d3a; --text: #e1e4ed;
    --muted: #8b8fa3; --accent: #6366f1; --accent2: #22d3ee; --red: #ef4444;
    --orange: #f97316; --green: #22c55e; --yellow: #eab308;
  }}
  * {{ margin:0; padding:0; box-sizing:border-box; }}
  body {{ background:var(--bg); color:var(--text); font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; padding:20px; }}
  h1 {{ font-size:1.8rem; margin-bottom:4px; }}
  .subtitle {{ color:var(--muted); margin-bottom:24px; font-size:0.9rem; }}
  .stats-grid {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(200px,1fr)); gap:16px; margin-bottom:24px; }}
  .stat-card {{ background:var(--card); border:1px solid var(--border); border-radius:12px; padding:20px; }}
  .stat-card .label {{ color:var(--muted); font-size:0.8rem; text-transform:uppercase; letter-spacing:0.05em; margin-bottom:6px; }}
  .stat-card .value {{ font-size:1.8rem; font-weight:700; }}
  .stat-card .value.red {{ color:var(--red); }}
  .stat-card .value.accent {{ color:var(--accent); }}
  .stat-card .value.cyan {{ color:var(--accent2); }}
  .stat-card .value.green {{ color:var(--green); }}
  .stat-card .value.orange {{ color:var(--orange); }}
  .charts-grid {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(500px,1fr)); gap:20px; margin-bottom:24px; }}
  .chart-card {{ background:var(--card); border:1px solid var(--border); border-radius:12px; padding:20px; }}
  .chart-card h3 {{ font-size:1rem; margin-bottom:12px; color:var(--text); }}
  .chart-card.full {{ grid-column:1/-1; }}
  canvas {{ max-height:320px; }}
  .table-wrap {{ overflow-x:auto; }}
  table {{ width:100%; border-collapse:collapse; font-size:0.85rem; }}
  th {{ text-align:left; padding:10px 12px; background:#12141c; color:var(--muted); font-weight:600; text-transform:uppercase; font-size:0.75rem; letter-spacing:0.05em; }}
  td {{ padding:10px 12px; border-top:1px solid var(--border); }}
  .cmd-cell {{ font-family:'SF Mono',Consolas,monospace; font-size:0.8rem; color:var(--accent2); word-break:break-all; max-width:500px; position:relative; }}
  .malware-row {{ cursor:pointer; transition:background 0.15s; }}
  .malware-row:hover {{ background:#22253a; }}
  .click-hint {{ display:inline-block; margin-left:8px; font-size:0.65rem; color:var(--muted); font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif; opacity:0.6; vertical-align:middle; }}
  .malware-row:hover .click-hint {{ color:var(--accent); opacity:1; }}
  .section-title {{ font-size:1.2rem; margin:32px 0 16px; padding-bottom:8px; border-bottom:1px solid var(--border); }}
  .heatmap-grid {{ display:grid; grid-template-columns:50px repeat(24,1fr); gap:2px; font-size:0.7rem; }}
  .heatmap-cell {{ aspect-ratio:1; border-radius:3px; display:flex; align-items:center; justify-content:center; font-size:0.6rem; color:rgba(255,255,255,0.7); }}
  .heatmap-label {{ display:flex; align-items:center; justify-content:flex-end; padding-right:6px; color:var(--muted); font-size:0.75rem; }}
  .heatmap-hour {{ display:flex; align-items:flex-end; justify-content:center; color:var(--muted); font-size:0.7rem; padding-bottom:4px; }}
  .insight-box {{ background:linear-gradient(135deg,#1e1b4b,#1a1d27); border:1px solid #4338ca; border-radius:12px; padding:20px; margin-bottom:24px; }}
  .insight-box h3 {{ color:var(--accent); margin-bottom:10px; }}
  .insight-box ul {{ list-style:none; }}
  .insight-box li {{ padding:4px 0; color:var(--text); font-size:0.9rem; }}
  .insight-box li::before {{ content:'\\2192  '; color:var(--accent); }}
  .modal-overlay {{ display:none; position:fixed; inset:0; background:rgba(0,0,0,0.7); z-index:1000; align-items:center; justify-content:center; padding:20px; }}
  .modal-overlay.active {{ display:flex; }}
  .modal {{ background:var(--card); border:1px solid var(--border); border-radius:12px; padding:24px; max-width:800px; width:100%; max-height:80vh; overflow-y:auto; position:relative; }}
  .modal h3 {{ margin-bottom:16px; font-size:1rem; }}
  .modal-cmd {{ font-family:'SF Mono',Consolas,monospace; font-size:0.85rem; color:var(--accent2); background:#12141c; padding:16px; border-radius:8px; word-break:break-all; white-space:pre-wrap; line-height:1.6; user-select:all; }}
  .modal-close {{ position:absolute; top:12px; right:16px; background:none; border:none; color:var(--muted); font-size:1.4rem; cursor:pointer; padding:4px 8px; border-radius:4px; }}
  .modal-close:hover {{ color:var(--text); background:#2a2d3a; }}
  .modal-copy {{ margin-top:12px; padding:6px 14px; background:var(--accent); color:#fff; border:none; border-radius:6px; font-size:0.8rem; cursor:pointer; }}
  .modal-copy:hover {{ opacity:0.85; }}
  @media(max-width:600px) {{
    .charts-grid {{ grid-template-columns:1fr; }}
    .stats-grid {{ grid-template-columns:repeat(2,1fr); }}
  }}
</style>
</head>
<body>
<h1>SSH Honeypot Dashboard</h1>
<p class="subtitle">Data from {m["date_start"]} to {m["date_end"]} &middot; {m["total_events"]:,} events captured</p>

<div class="stats-grid">
  <div class="stat-card"><div class="label">Total Events</div><div class="value accent">{m["total_events"]:,}</div></div>
  <div class="stat-card"><div class="label">Unique Source IPs</div><div class="value red">{m["unique_ips"]:,}</div></div>
  <div class="stat-card"><div class="label">Unique Usernames</div><div class="value cyan">{m["unique_users"]}</div></div>
  <div class="stat-card"><div class="label">Login Attempts</div><div class="value orange">{m["login_count"]:,}</div></div>
  <div class="stat-card"><div class="label">Commands Typed</div><div class="value green">{m["typed_count"]:,}</div></div>
  <div class="stat-card"><div class="label">Sessions</div><div class="value accent">{m["total_sessions"]:,}</div></div>
  <div class="stat-card"><div class="label">Avg Session Duration</div><div class="value cyan">{m["avg_session_dur"]}s</div></div>
  <div class="stat-card"><div class="label">Malware Attempts</div><div class="value red">{m["malware_attempts"]}</div></div>
</div>

<div class="insight-box">
  <h3>Key Insights</h3>
  <ul>{insights_html}</ul>
</div>

<div class="charts-grid">
  <div class="chart-card full"><h3>Daily Event Timeline (Login vs Commands)</h3><canvas id="dailyChart"></canvas></div>
  <div class="chart-card"><h3>Monthly Trend</h3><canvas id="monthlyChart"></canvas></div>
  <div class="chart-card"><h3>Hourly Activity (UTC)</h3><canvas id="hourlyChart"></canvas></div>
  <div class="chart-card"><h3>Top 15 Usernames</h3><canvas id="usersChart"></canvas></div>
  <div class="chart-card"><h3>Top 15 Source IPs</h3><canvas id="ipsChart"></canvas></div>
  <div class="chart-card"><h3>Command Categories</h3><canvas id="catChart"></canvas></div>
  <div class="chart-card"><h3>Event Type Distribution</h3><canvas id="typeChart"></canvas></div>
  <div class="chart-card"><h3>Top Commands Executed</h3><canvas id="cmdsChart"></canvas></div>
  <div class="chart-card"><h3>Activity by Day of Week</h3><canvas id="dowChart"></canvas></div>
  <div class="chart-card"><h3>Login Attempts by Username</h3><canvas id="loginChart"></canvas></div>
</div>

<h2 class="section-title">Activity Heatmap (Day of Week &times; Hour UTC)</h2>
<div class="chart-card full" style="margin-bottom:24px;">
  <div class="heatmap-grid" id="heatmap"><div></div></div>
</div>

<h2 class="section-title">Top Malware / Download Attempts</h2>
<div class="chart-card full">
  <div class="table-wrap">
    <table>
      <thead><tr><th>Command</th><th>Count</th><th>Source IPs</th></tr></thead>
      <tbody>{malware_rows}</tbody>
    </table>
  </div>
</div>

<div class="modal-overlay" id="cmdModal">
  <div class="modal">
    <button class="modal-close" id="modalClose">&times;</button>
    <h3>Full Malware Command</h3>
    <div class="modal-cmd" id="modalCmdText"></div>
    <button class="modal-copy" id="modalCopy">Copy to clipboard</button>
  </div>
</div>

<script>
const darkOpts = {{
  responsive:true, maintainAspectRatio:true,
  plugins:{{ legend:{{ labels:{{ color:'#8b8fa3' }} }} }},
  scales:{{
    x:{{ ticks:{{ color:'#8b8fa3', maxRotation:45 }}, grid:{{ color:'#2a2d3a' }} }},
    y:{{ ticks:{{ color:'#8b8fa3' }}, grid:{{ color:'#2a2d3a' }} }}
  }}
}};
const barOpts = (h) => ({{ ...darkOpts, indexAxis:h?'y':'x', plugins:{{ legend:{{ display:false }} }} }});
const pieColors = ['#6366f1','#22d3ee','#f97316','#ef4444','#22c55e','#eab308','#a855f7','#ec4899'];

new Chart(document.getElementById('dailyChart'), {{
  type:'bar',
  data:{{ labels:{j(m["daily_labels"])}, datasets:[
    {{ label:'Logins', data:{j(m["daily_login"])}, backgroundColor:'#ef4444aa', borderRadius:2 }},
    {{ label:'Commands', data:{j(m["daily_typed"])}, backgroundColor:'#6366f1aa', borderRadius:2 }}
  ] }},
  options:{{ ...darkOpts, scales:{{ ...darkOpts.scales, x:{{ ...darkOpts.scales.x, stacked:true }}, y:{{ ...darkOpts.scales.y, stacked:true }} }} }}
}});

new Chart(document.getElementById('monthlyChart'), {{
  type:'bar',
  data:{{ labels:{j(m["monthly_labels"])}, datasets:[{{ data:{j(m["monthly_values"])}, backgroundColor:'#6366f1', borderRadius:4 }}] }},
  options:barOpts(false)
}});

new Chart(document.getElementById('hourlyChart'), {{
  type:'line',
  data:{{ labels:{j(m["hourly_labels"])}, datasets:[{{ data:{j(m["hourly_values"])}, borderColor:'#22d3ee', backgroundColor:'#22d3ee22', fill:true, tension:0.3 }}] }},
  options:{{ ...darkOpts, plugins:{{ legend:{{ display:false }} }} }}
}});

new Chart(document.getElementById('usersChart'), {{
  type:'bar',
  data:{{ labels:{j(m["top_users"]["labels"])}, datasets:[{{ data:{j(m["top_users"]["values"])}, backgroundColor:'#f97316' }}] }},
  options:barOpts(true)
}});

new Chart(document.getElementById('ipsChart'), {{
  type:'bar',
  data:{{ labels:{j(m["top_ips"]["labels"])}, datasets:[{{ data:{j(m["top_ips"]["values"])}, backgroundColor:'#ef4444' }}] }},
  options:barOpts(true)
}});

new Chart(document.getElementById('catChart'), {{
  type:'doughnut',
  data:{{ labels:{j(m["categories"]["labels"])}, datasets:[{{ data:{j(m["categories"]["values"])}, backgroundColor:pieColors }}] }},
  options:{{ responsive:true, plugins:{{ legend:{{ position:'right', labels:{{ color:'#8b8fa3', padding:12 }} }} }} }}
}});

new Chart(document.getElementById('typeChart'), {{
  type:'doughnut',
  data:{{ labels:{j(m["event_types"]["labels"])}, datasets:[{{ data:{j(m["event_types"]["values"])}, backgroundColor:['#6366f1','#22d3ee','#22c55e'] }}] }},
  options:{{ responsive:true, plugins:{{ legend:{{ position:'right', labels:{{ color:'#8b8fa3', padding:12 }} }} }} }}
}});

new Chart(document.getElementById('cmdsChart'), {{
  type:'bar',
  data:{{ labels:{j(m["top_commands"]["labels"])}, datasets:[{{ data:{j(m["top_commands"]["values"])}, backgroundColor:'#22c55e' }}] }},
  options:barOpts(true)
}});

new Chart(document.getElementById('dowChart'), {{
  type:'bar',
  data:{{ labels:{j(m["dow_labels"])}, datasets:[{{ data:{j(m["dow_values"])}, backgroundColor:'#a855f7' }}] }},
  options:barOpts(false)
}});

new Chart(document.getElementById('loginChart'), {{
  type:'bar',
  data:{{ labels:{j(m["login_users"]["labels"])}, datasets:[{{ data:{j(m["login_users"]["values"])}, backgroundColor:'#eab308' }}] }},
  options:barOpts(true)
}});

// Heatmap
const heatData = {j(m["heatmap"])};
const dowNames = ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'];
const grid = document.getElementById('heatmap');
for(let h=0;h<24;h++) {{
  const el = document.createElement('div');
  el.className = 'heatmap-hour';
  el.textContent = h.toString().padStart(2,'0');
  grid.appendChild(el);
}}
const maxVal = Math.max(...heatData.map(d=>d.v));
for(let d=0;d<7;d++) {{
  const lbl = document.createElement('div');
  lbl.className = 'heatmap-label';
  lbl.textContent = dowNames[d];
  grid.appendChild(lbl);
  for(let h=0;h<24;h++) {{
    const cell = document.createElement('div');
    cell.className = 'heatmap-cell';
    const entry = heatData.find(e=>e.x===h&&e.y===d);
    const v = entry?entry.v:0;
    const intensity = v/maxVal;
    cell.style.background = `rgba(99,102,241,${{Math.max(0.05,intensity).toFixed(2)}})`;
    if(v>100) cell.textContent = v>999?(v/1000).toFixed(1)+'k':v;
    cell.title = `${{dowNames[d]}} ${{h.toString().padStart(2,'0')}}:00 — ${{v}} events`;
    grid.appendChild(cell);
  }}
}}

// Modal
document.addEventListener('DOMContentLoaded', () => {{
  const overlay = document.getElementById('cmdModal');
  const modalCmd = document.getElementById('modalCmdText');
  const modalClose = document.getElementById('modalClose');
  const modalCopy = document.getElementById('modalCopy');
  document.querySelectorAll('.malware-row').forEach(row => {{
    row.addEventListener('click', () => {{
      modalCmd.textContent = row.getAttribute('data-full');
      overlay.classList.add('active');
    }});
  }});
  modalClose.addEventListener('click', () => overlay.classList.remove('active'));
  overlay.addEventListener('click', (e) => {{ if(e.target===overlay) overlay.classList.remove('active'); }});
  document.addEventListener('keydown', (e) => {{ if(e.key==='Escape') overlay.classList.remove('active'); }});
  modalCopy.addEventListener('click', () => {{
    navigator.clipboard.writeText(modalCmd.textContent).then(() => {{
      modalCopy.textContent = 'Copied!';
      setTimeout(() => modalCopy.textContent = 'Copy to clipboard', 1500);
    }});
  }});
}});
</script>
</body>
</html>"""


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <metrics.json> <output.html>", file=sys.stderr)
        sys.exit(1)

    metrics_path = sys.argv[1]
    output_path = sys.argv[2]

    with open(metrics_path, "r") as f:
        metrics = json.load(f)

    dashboard = render(metrics)
    with open(output_path, "w") as f:
        f.write(dashboard)

    print(f"Dashboard written to {output_path}", file=sys.stderr)


if __name__ == "__main__":
    main()
