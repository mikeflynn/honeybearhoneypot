package gui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/log/v2"
	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func adminStatsTab() *fyne.Container {
	content := container.NewStack()

	// Define the views
	views := map[string]func() *fyne.Container{
		"Users 30d":     statsUsersGraphTab,
		"Cmds 30d":      statsCommandsGraphTab,
		"Top Cmds":      statsTopCommandsTab,
		"Top Users":     statsTopUsersTab,
		"Top Passwords": statsTopPasswordsTab,
		"Recent":        statsRecentCommandsTab,
	}

	// Order for the dropdown
	options := []string{"Users 30d", "Cmds 30d", "Top Cmds", "Top Users", "Top Passwords", "Recent"}

	selector := widget.NewSelect(options, func(s string) {
		if fn, ok := views[s]; ok {
			content.Objects = []fyne.CanvasObject{fn()}
			content.Refresh()
		}
	})

	// Initial selection
	selector.Selected = options[0]
	content.Objects = []fyne.CanvasObject{views[options[0]]()}

	return container.NewBorder(
		container.NewPadded(selector), // Header / Navigation
		nil, nil, nil,
		content, // Content area
	)
}

func statsUsersGraphTab() *fyne.Container {
	content := container.NewStack()

	go func() {
		// Users per day (last 30 days)
		rows, err := entity.EventCountQuery(
			`SELECT
				strftime('%Y-%m-%d', timestamp) as day,
				COUNT(DISTINCT user) as count
			FROM events
			WHERE type='login'
			AND timestamp >= datetime('now', '-30 days')
			GROUP BY day
			ORDER BY day ASC`,
		)

		if err != nil {
			log.Error("Error querying user stats", "error", err)
			return
		}

		chart := makeLineChart(rows)
		fyne.Do(func() {
			content.Objects = []fyne.CanvasObject{chart}
			content.Refresh()
		})
	}()

	return content
}

func statsCommandsGraphTab() *fyne.Container {
	content := container.NewStack()

	go func() {
		rows, err := entity.EventCountQuery(
			`SELECT
				strftime('%Y-%m-%d', timestamp) as day,
				COUNT(*) as count
			FROM events
			WHERE type='typed'
			AND timestamp >= datetime('now', '-30 days')
			GROUP BY day
			ORDER BY day ASC`,
		)

		if err != nil {
			log.Error("Error querying command stats", "error", err)
			return
		}

		chart := makeLineChart(rows)
		fyne.Do(func() {
			content.Objects = []fyne.CanvasObject{chart}
			content.Refresh()
		})
	}()

	return content
}

func statsTopCommandsTab() *fyne.Container {
	return createListTab(func() ([]string, error) {
		rows, err := entity.EventCountQuery(
			`SELECT
				action,
				count(*) AS total
			FROM events
			WHERE events.type = 'typed'
			GROUP BY events.action
			ORDER by count(*) DESC
			LIMIT 100`,
		)
		if err != nil {
			return nil, err
		}

		data := []string{}
		for i, e := range rows {
			data = append(data, fmt.Sprintf("%d. %s (%d)", i+1, e.Value, e.Count))
		}
		return data, nil
	})
}

func statsTopUsersTab() *fyne.Container {
	return createListTab(func() ([]string, error) {
		rows, err := entity.EventCountQuery(
			`SELECT
				user || ' (' || CASE WHEN instr(host, ':') > 0 THEN substr(host, 1, instr(host, ':') - 1) ELSE host END || ')',
				count(*) AS total
			FROM events
			WHERE events.type = 'login'
			GROUP BY user, CASE WHEN instr(host, ':') > 0 THEN substr(host, 1, instr(host, ':') - 1) ELSE host END
			ORDER by count(*) DESC
			LIMIT 100`,
		)
		if err != nil {
			return nil, err
		}

		data := []string{}
		for i, e := range rows {
			data = append(data, fmt.Sprintf("%d. %s: %d", i+1, e.Value, e.Count))
		}
		return data, nil
	})
}

func statsTopPasswordsTab() *fyne.Container {
	return createListTab(func() ([]string, error) {
		rows, err := entity.EventCountQuery(
			`SELECT
				action,
				count(*) AS total
			FROM events
			WHERE events.type = 'login'
			AND action != ''
			AND action != 'Logged in!'
			AND action != 'Logged in via exec'
			GROUP BY action
			ORDER by count(*) DESC
			LIMIT 100`,
		)
		if err != nil {
			return nil, err
		}

		data := []string{}
		for i, e := range rows {
			data = append(data, fmt.Sprintf("%d. %s (%d)", i+1, e.Value, e.Count))
		}
		return data, nil
	})
}

func statsRecentCommandsTab() *fyne.Container {
	return createListTab(func() ([]string, error) {
		events, err := entity.EventQuery(
			`SELECT *
			 FROM events
			 WHERE app = "ssh"
			 ORDER by timestamp DESC
			 LIMIT 100`,
		)
		if err != nil {
			return nil, err
		}

		data := []string{}
		// Attempt to load local timezone, fallback to UTC
		tz, err := time.LoadLocation("Local")
		if err != nil {
			tz = time.UTC
		}

		for _, e := range events {
			timestamp := e.Timestamp.In(tz).Format("01/02 15:04")
			data = append(data, fmt.Sprintf("[%s] %s (%s)\n%s", timestamp, e.User, e.Host, e.Action))
		}
		return data, nil
	})
}

// Helper to create a tab with a list
func createListTab(dataLoader func() ([]string, error)) *fyne.Container {
	listData := []string{"Loading..."}
	listBinding := binding.NewStringList()
	listBinding.Set(listData)

	list := widget.NewListWithData(
		listBinding,
		func() fyne.CanvasObject {
			t1 := canvas.NewText("template", theme.Color(theme.ColorNameForeground))
			t1.TextSize = 16
			t1.TextStyle = fyne.TextStyle{Monospace: true}
			t2 := canvas.NewText("template", theme.Color(theme.ColorNameForeground))
			t2.TextSize = 16
			t2.TextStyle = fyne.TextStyle{Monospace: true}

			return container.NewVBox(t1, t2)
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			s, _ := i.(binding.String).Get()
			box := o.(*fyne.Container)
			t1 := box.Objects[0].(*canvas.Text)
			t2 := box.Objects[1].(*canvas.Text)

			lines := strings.Split(s, "\n")
			t1.Text = lines[0]
			t1.Color = theme.Color(theme.ColorNameForeground)
			if len(lines) > 1 {
				t2.Text = lines[1]
				t2.Color = theme.Color(theme.ColorNamePrimary)
				t2.Show()
			} else {
				t2.Text = ""
				t2.Hide()
			}
			t1.Refresh()
			t2.Refresh()
			box.Refresh()
		},
	)

	go func() {
		data, err := dataLoader()
		if err != nil {
			log.Error("Error loading list data", "error", err)
			listBinding.Set([]string{"Error loading data"})
			return
		}
		listBinding.Set(data)
	}()

	return container.NewStack(list)
}

// Simple Line Chart Implementation
func makeLineChart(data []*entity.EventCount) fyne.CanvasObject {
	// 1. Fill in gaps (last 30 days)
	daysMap := make(map[string]int)
	for _, d := range data {
		daysMap[d.Value] = d.Count
	}

	points := []float32{}
	labels := []string{}
	maxVal := 0

	now := time.Now()
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		val := daysMap[key]
		if val > maxVal {
			maxVal = val
		}
		points = append(points, float32(val))

		// Add labels for every 5 days or so
		if i%5 == 0 {
			labels = append(labels, d.Format("Jan 02"))
		} else {
			labels = append(labels, "")
		}
	}

	if maxVal < 5 {
		maxVal = 5
	}

	// 2. Create Custom Widget for Drawing

	chart := newSimpleChart(points, labels, float32(maxVal))

	return container.NewPadded(
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Peak: %d", maxVal)),
			container.NewPadded(chart),
		),
	)
}

func newSimpleChart(points []float32, labels []string, maxVal float32) *simpleChart {
	chart := &simpleChart{points: points, labels: labels, maxVal: maxVal}
	chart.ExtendBaseWidget(chart)
	return chart
}

type simpleChart struct {
	widget.BaseWidget
	points []float32
	labels []string
	maxVal float32
}

func (s *simpleChart) CreateRenderer() fyne.WidgetRenderer {
	return &simpleChartRenderer{chart: s}
}

type simpleChartRenderer struct {
	chart   *simpleChart
	objects []fyne.CanvasObject
}

func (r *simpleChartRenderer) Layout(size fyne.Size) {
	r.objects = nil
	if len(r.chart.points) < 2 {
		return
	}

	padding := float32(10)
	bottomPadding := float32(30)
	width := size.Width
	height := size.Height
	availableHeight := height - padding - bottomPadding
	stepX := width / float32(len(r.chart.points)-1)

	// Draw Axes
	// ... (Skipping for simplicity, just sparkline)

	var lastX, lastY float32

	for i, val := range r.chart.points {
		x := float32(i) * stepX
		// Invert Y because 0 is at top
		y := availableHeight + padding - (val / r.chart.maxVal * availableHeight)

		if i > 0 {
			line := canvas.NewLine(theme.Color(theme.ColorNameForeground))
			line.StrokeWidth = 2
			line.Position1 = fyne.NewPos(lastX, lastY)
			line.Position2 = fyne.NewPos(x, y)
			r.objects = append(r.objects, line)
		}

		// Draw dot
		dot := canvas.NewCircle(theme.Color(theme.ColorNameForeground))
		dot.Resize(fyne.NewSize(4, 4))
		dot.Move(fyne.NewPos(x-2, y-2))
		r.objects = append(r.objects, dot)

		// Draw Label
		if i < len(r.chart.labels) && r.chart.labels[i] != "" {
			label := canvas.NewText(r.chart.labels[i], theme.Color(theme.ColorNameForeground))
			label.TextSize = 10
			label.Alignment = fyne.TextAlignCenter
			// Center the text on the X coordinate
			label.Move(fyne.NewPos(x-15, height-bottomPadding+5))
			r.objects = append(r.objects, label)
		}

		lastX = x
		lastY = y
	}
}

func (r *simpleChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(300, 200)
}

func (r *simpleChartRenderer) Refresh() {
	r.Layout(r.chart.Size())
	canvas.Refresh(r.chart)
}

func (r *simpleChartRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *simpleChartRenderer) Destroy() {}
