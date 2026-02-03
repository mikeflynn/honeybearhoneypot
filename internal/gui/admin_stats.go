package gui

import (
	"fmt"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
)

func adminStatsTab() *fyne.Container {
	content := container.NewStack()

	// Define the views
	views := map[string]func() *fyne.Container{
		"Users 30d": statsUsersGraphTab,
		"Cmds 30d":  statsCommandsGraphTab,
		"Top Cmds":  statsTopCommandsTab,
		"Top Users": statsTopUsersTab,
		"Recent":    statsRecentCommandsTab,
	}

	// Order for the dropdown
	options := []string{"Users 30d", "Cmds 30d", "Top Cmds", "Top Users", "Recent"}

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
			timestamp := e.Timestamp.In(tz).Format("2006-01-02 15:04:05")
			data = append(data, fmt.Sprintf("[%s] %s (%s) > %s", timestamp, e.User, e.Host, e.Action))
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
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
			o.(*widget.Label).TextStyle = fyne.TextStyle{Monospace: true}
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

	if maxVal == 0 {
		maxVal = 1 // Avoid division by zero
	}

	// 2. Create Custom Widget for Drawing

	return container.NewPadded(
		container.NewVBox(
			//widget.NewLabel(fmt.Sprintf("Peak: %d", maxVal)),
			container.NewPadded(&simpleChart{points: points, maxVal: float32(maxVal)}),
		),
	)
}

type simpleChart struct {
	widget.BaseWidget
	points []float32
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

	width := size.Width
	height := size.Height
	stepX := width / float32(len(r.chart.points)-1)

	// Draw Axes
	// ... (Skipping for simplicity, just sparkline)

	var lastX, lastY float32

	for i, val := range r.chart.points {
		x := float32(i) * stepX
		// Invert Y because 0 is at top
		y := height - (val / r.chart.maxVal * height)

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

		lastX = x
		lastY = y
	}
}

func (r *simpleChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 150)
}

func (r *simpleChartRenderer) Refresh() {
	r.Layout(r.chart.Size())
	canvas.Refresh(r.chart)
}

func (r *simpleChartRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *simpleChartRenderer) Destroy() {}
