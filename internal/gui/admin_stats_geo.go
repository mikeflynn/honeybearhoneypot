package gui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"net"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"charm.land/log/v2"
	"github.com/mikeflynn/honeybearhoneypot/internal/entity"
	"github.com/mikeflynn/honeybearhoneypot/internal/geo"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui/assets"
)

// pin is one country's worth of login events.
type pin struct {
	Lat, Lon    float64
	Hits        int
	Country     string
	CountryCode string // ISO2; merge key
}

// extractIP returns the IP portion of a host string, handling both
// "ip:port" (IPv4) and "[ip]:port" (IPv6) forms, as well as a bare IP.
func extractIP(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// projectLatLon converts WGS-84 lat/lon to (x,y) within an equirectangular
// canvas of size (w,h). lat: +90 (north pole) -> y=0, -90 -> y=h.
// lon: -180 -> x=0, +180 -> x=w.
func projectLatLon(lat, lon float64, w, h float32) (float32, float32) {
	x := float32((lon+180.0)/360.0) * w
	y := float32((90.0-lat)/180.0) * h
	return x, y
}

func adminMapTab() *fyne.Container {
	rangeOpts := []string{"30 days", "90 days", "120 days"}
	rangeDays := map[string]int{"30 days": 30, "90 days": 90, "120 days": 120}

	mapArea := container.NewStack()

	rebuild := func(days int) {
		mapArea.Objects = []fyne.CanvasObject{widget.NewLabel("Loading map...")}
		mapArea.Refresh()

		go func() {
			pins, err := loadGeoPins(days)
			if err != nil {
				log.Error("geo map load failed", "err", err)
				fyne.Do(func() {
					mapArea.Objects = []fyne.CanvasObject{widget.NewLabel("Error loading map: " + err.Error())}
					mapArea.Refresh()
				})
				return
			}
			mw := newMapWidget(pins)
			fyne.Do(func() {
				mapArea.Objects = []fyne.CanvasObject{mw}
				mapArea.Refresh()
			})
		}()
	}

	sel := widget.NewSelect(rangeOpts, func(s string) {
		rebuild(rangeDays[s])
	})
	sel.Selected = rangeOpts[0]

	rebuild(30)

	return container.NewBorder(
		sel,
		nil, nil, nil,
		mapArea,
	)
}

// loadGeoPins runs the SQL aggregation, resolves each IP to a country
// centroid, and merges all hits per ISO2 into one pin.
func loadGeoPins(days int) ([]pin, error) {
	since := fmt.Sprintf("-%d days", days)
	rows, err := entity.EventCountQuery(
		`SELECT host, COUNT(*) AS hits
		 FROM events
		 WHERE type='login'
		   AND timestamp >= datetime('now', ?)
		 GROUP BY host`,
		since,
	)
	if err != nil {
		return nil, err
	}

	bucket := map[string]*pin{} // keyed by ISO2

	for _, r := range rows {
		ip := extractIP(r.Value)
		if ip == "" {
			continue
		}
		g, err := geo.Lookup(ip)
		if err != nil || g == nil {
			continue
		}
		p, ok := bucket[g.CountryCode]
		if !ok {
			p = &pin{
				Lat:         g.Lat,
				Lon:         g.Lon,
				Country:     g.Country,
				CountryCode: g.CountryCode,
			}
			bucket[g.CountryCode] = p
		}
		p.Hits += r.Count
	}

	out := make([]pin, 0, len(bucket))
	for _, p := range bucket {
		out = append(out, *p)
	}
	return out, nil
}

// --- mapWidget: draws worldmap.png and overlays clickable pins ---

type mapWidget struct {
	widget.BaseWidget
	pins []pin
}

func newMapWidget(pins []pin) *mapWidget {
	m := &mapWidget{pins: pins}
	m.ExtendBaseWidget(m)
	return m
}

func (m *mapWidget) CreateRenderer() fyne.WidgetRenderer {
	imgData, err := assets.Images.ReadFile("worldmap.png")
	if err != nil {
		log.Error("worldmap asset missing", "err", err)
		return widget.NewSimpleRenderer(widget.NewLabel("worldmap.png missing"))
	}
	src, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		log.Error("worldmap decode failed", "err", err)
		return widget.NewSimpleRenderer(widget.NewLabel("worldmap decode failed"))
	}
	bg := canvas.NewImageFromImage(src)
	bg.FillMode = canvas.ImageFillContain
	return &mapRenderer{widget: m, bg: bg}
}

// Tapped maps a tap on the widget to the nearest pin (within 12 px) and shows
// a dialog. Implements fyne.Tappable.
func (m *mapWidget) Tapped(ev *fyne.PointEvent) {
	size := m.Size()
	cw, ch := containedSize(size)
	originX := (size.Width - cw) / 2
	originY := (size.Height - ch) / 2

	const tapRadius = 12.0
	var hit *pin
	bestDist := float32(tapRadius * tapRadius)
	for i := range m.pins {
		px, py := projectLatLon(m.pins[i].Lat, m.pins[i].Lon, cw, ch)
		px += originX
		py += originY
		dx := ev.Position.X - px
		dy := ev.Position.Y - py
		d := dx*dx + dy*dy
		if d <= bestDist {
			bestDist = d
			hit = &m.pins[i]
		}
	}
	if hit == nil {
		return
	}
	count := canvas.NewText(fmt.Sprintf("%d", hit.Hits), theme.Color(theme.ColorNamePrimary))
	count.TextSize = 64
	count.TextStyle = fyne.TextStyle{Bold: true}
	count.Alignment = fyne.TextAlignCenter

	label := canvas.NewText("connections", theme.Color(theme.ColorNameForeground))
	label.TextSize = 14
	label.Alignment = fyne.TextAlignCenter

	dialog.ShowCustom(hit.Country, "Close",
		container.NewVBox(count, label), w)
}

// containedSize returns the size of a 2:1 image when rendered with
// ImageFillContain inside the given widget size.
func containedSize(s fyne.Size) (float32, float32) {
	const aspect = 2.0
	if s.Width/s.Height > aspect {
		h := s.Height
		w := h * aspect
		return w, h
	}
	w := s.Width
	h := w / aspect
	return w, h
}

type mapRenderer struct {
	widget  *mapWidget
	bg      *canvas.Image
	objects []fyne.CanvasObject
}

func (r *mapRenderer) Layout(size fyne.Size) {
	cw, ch := containedSize(size)
	originX := (size.Width - cw) / 2
	originY := (size.Height - ch) / 2

	r.bg.Resize(size)

	objs := []fyne.CanvasObject{r.bg}

	maxHits := 1
	for _, p := range r.widget.pins {
		if p.Hits > maxHits {
			maxHits = p.Hits
		}
	}

	for _, p := range r.widget.pins {
		px, py := projectLatLon(p.Lat, p.Lon, cw, ch)
		px += originX
		py += originY

		radius := float32(4)
		h := p.Hits
		for h > 1 && radius < 14 {
			h /= 2
			radius += 2
		}

		dot := canvas.NewCircle(theme.Color(theme.ColorNameError))
		dot.StrokeColor = theme.Color(theme.ColorNameForeground)
		dot.StrokeWidth = 1
		dot.Resize(fyne.NewSize(radius*2, radius*2))
		dot.Move(fyne.NewPos(px-radius, py-radius))
		objs = append(objs, dot)
	}

	r.objects = objs
}

func (r *mapRenderer) MinSize() fyne.Size           { return fyne.NewSize(600, 320) }
func (r *mapRenderer) Refresh()                     { r.Layout(r.widget.Size()); canvas.Refresh(r.widget) }
func (r *mapRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *mapRenderer) Destroy()                     {}
