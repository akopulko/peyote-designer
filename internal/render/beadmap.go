package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/kostya/peyote-designer/internal/app"
	"github.com/kostya/peyote-designer/internal/model"
)

const (
	baseBeadWidth  = 16
	baseBeadHeight = 32
	baseGap        = 4
)

type Metrics struct {
	BeadWidth  int
	BeadHeight int
	Gap        int
}

func ComputeMetrics(zoom float32) Metrics {
	return Metrics{
		BeadWidth:  int(math.Round(float64(baseBeadWidth) * float64(zoom))),
		BeadHeight: int(math.Round(float64(baseBeadHeight) * float64(zoom))),
		Gap:        int(math.Max(1, math.Round(float64(baseGap)*float64(zoom)))),
	}
}

type BeadMap struct {
	widget.BaseWidget
	controller *app.Controller
	raster     *canvas.Raster
}

func NewBeadMap(controller *app.Controller) *BeadMap {
	m := &BeadMap{controller: controller}
	m.raster = canvas.NewRaster(m.render)
	m.ExtendBaseWidget(m)
	return m
}

func (m *BeadMap) Tapped(event *fyne.PointEvent) {
	row, col, ok := m.HitTest(event.Position)
	if !ok {
		return
	}
	_ = m.controller.ActivateBead(row, col)
}

func (m *BeadMap) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.raster)
}

func (m *BeadMap) MinSize() fyne.Size {
	session := m.controller.Session()
	metrics := ComputeMetrics(session.Zoom)
	width := session.Document.Canvas.Width*(metrics.BeadWidth+metrics.Gap) + metrics.Gap
	height := session.Document.Canvas.Height*(metrics.BeadHeight+metrics.Gap) + metrics.Gap
	return fyne.NewSize(float32(width), float32(height))
}

func (m *BeadMap) HitTest(position fyne.Position) (int, int, bool) {
	session := m.controller.Session()
	metrics := ComputeMetrics(session.Zoom)
	col := int((position.X - float32(metrics.Gap)) / float32(metrics.BeadWidth+metrics.Gap))
	row := int((position.Y - float32(metrics.Gap)) / float32(metrics.BeadHeight+metrics.Gap))
	if row < 0 || col < 0 || row >= session.Document.Canvas.Height || col >= session.Document.Canvas.Width {
		return 0, 0, false
	}
	return row, col, true
}

func (m *BeadMap) render(width, height int) image.Image {
	session := m.controller.Session()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{R: 245, G: 243, B: 238, A: 255}}, image.Point{}, draw.Src)

	metrics := ComputeMetrics(session.Zoom)
	for row := 0; row < session.Document.Canvas.Height; row++ {
		for col := 0; col < session.Document.Canvas.Width; col++ {
			x := metrics.Gap + col*(metrics.BeadWidth+metrics.Gap)
			y := metrics.Gap + row*(metrics.BeadHeight+metrics.Gap)
			rect := image.Rect(x, y, x+metrics.BeadWidth, y+metrics.BeadHeight)
			fill := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			bead := session.Document.Beads[row][col]
			if bead.ColorID != "" {
				if paletteColor, ok := session.Document.PaletteColorByID(bead.ColorID); ok {
					fill = parseHexColor(paletteColor.Hex)
				}
			}
			if session.Selection.Mode == model.SelectionRow && session.Selection.Index == row {
				fill = lighten(fill, 18)
			}
			if session.Selection.Mode == model.SelectionColumn && session.Selection.Index == col {
				fill = lighten(fill, 18)
			}
			fillRect(img, rect, fill)
			strokeRect(img, rect, color.NRGBA{R: 80, G: 80, B: 80, A: 255})
			if bead.Completed {
				drawCross(img, rect, color.NRGBA{R: 20, G: 20, B: 20, A: 255})
			}
		}
	}
	return img
}

func fillRect(img *image.RGBA, rect image.Rectangle, fill color.NRGBA) {
	draw.Draw(img, rect, &image.Uniform{C: fill}, image.Point{}, draw.Src)
}

func strokeRect(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, stroke)
		img.Set(x, rect.Max.Y-1, stroke)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, stroke)
		img.Set(rect.Max.X-1, y, stroke)
	}
}

func drawCross(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA) {
	steps := min(rect.Dx(), rect.Dy())
	for i := 1; i < steps-1; i++ {
		img.Set(rect.Min.X+i, rect.Min.Y+i, stroke)
		img.Set(rect.Max.X-1-i, rect.Min.Y+i, stroke)
	}
}

func parseHexColor(value string) color.NRGBA {
	value = strings.TrimPrefix(model.NormalizeHex(value), "#")
	if len(value) != 6 {
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}
	red, _ := strconv.ParseUint(value[0:2], 16, 8)
	green, _ := strconv.ParseUint(value[2:4], 16, 8)
	blue, _ := strconv.ParseUint(value[4:6], 16, 8)
	return color.NRGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: 255}
}

func lighten(in color.NRGBA, delta uint8) color.NRGBA {
	return color.NRGBA{
		R: minUint8(255, in.R+delta),
		G: minUint8(255, in.G+delta),
		B: minUint8(255, in.B+delta),
		A: in.A,
	}
}

func minUint8(limit, value uint8) uint8 {
	if value > limit {
		return limit
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
