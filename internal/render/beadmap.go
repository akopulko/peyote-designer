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
	baseBeadHeight = 24
	baseGap        = 4
)

type Metrics struct {
	BeadWidth  int
	BeadHeight int
	Gap        int
}

func ComputeMetrics(zoom float32) Metrics {
	return computeMetricsScaled(zoom, 1)
}

func computeMetricsScaled(zoom float32, scale float32) Metrics {
	return Metrics{
		BeadWidth:  max(1, int(math.Round(float64(baseBeadWidth)*float64(zoom)*float64(scale)))),
		BeadHeight: max(1, int(math.Round(float64(baseBeadHeight)*float64(zoom)*float64(scale)))),
		Gap:        max(1, int(math.Round(float64(baseGap)*float64(zoom)*float64(scale)))),
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
	if session.Document == nil {
		return fyne.NewSize(320, 240)
	}
	metrics := ComputeMetrics(session.Zoom)
	width := session.Document.Canvas.Width*(metrics.BeadWidth+metrics.Gap) + metrics.Gap
	height := session.Document.Canvas.Height*(metrics.BeadHeight+metrics.Gap) + metrics.Gap
	return fyne.NewSize(float32(width), float32(height))
}

func (m *BeadMap) HitTest(position fyne.Position) (int, int, bool) {
	session := m.controller.Session()
	if session.Document == nil {
		return 0, 0, false
	}
	metrics := ComputeMetrics(session.Zoom)
	strideX := float32(metrics.BeadWidth + metrics.Gap)
	strideY := float32(metrics.BeadHeight + metrics.Gap)
	offsetX := position.X - float32(metrics.Gap)
	offsetY := position.Y - float32(metrics.Gap)
	if offsetX < 0 || offsetY < 0 {
		return 0, 0, false
	}
	col := int(offsetX / strideX)
	row := int(offsetY / strideY)
	if row < 0 || col < 0 || row >= session.Document.Canvas.Height || col >= session.Document.Canvas.Width {
		return 0, 0, false
	}
	localX := offsetX - float32(col)*strideX
	localY := offsetY - float32(row)*strideY
	if localX > float32(metrics.BeadWidth) || localY > float32(metrics.BeadHeight) {
		return 0, 0, false
	}
	return row, col, true
}

func (m *BeadMap) render(width, height int) image.Image {
	session := m.controller.Session()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{R: 245, G: 243, B: 238, A: 255}}, image.Point{}, draw.Src)
	if session.Document == nil {
		return img
	}

	scale := float32(1)
	size := m.Size()
	if size.Width > 0 && size.Height > 0 {
		scaleX := float32(width) / size.Width
		scaleY := float32(height) / size.Height
		scale = (scaleX + scaleY) / 2
	}
	metrics := computeMetricsScaled(session.Zoom, scale)
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
			fillRect(img, rect, fill)
			strokeRect(img, rect, color.NRGBA{R: 80, G: 80, B: 80, A: 255})
			if session.Selection.Mode == model.SelectionRow && session.Selection.Index == row {
				strokeRectThickness(img, rect, color.NRGBA{R: 198, G: 40, B: 40, A: 255}, 3)
			}
			if session.Selection.Mode == model.SelectionColumn && session.Selection.Index == col {
				strokeRectThickness(img, rect, color.NRGBA{R: 198, G: 40, B: 40, A: 255}, 3)
			}
			if bead.Completed {
				drawCross(img, rect, crossColor(fill))
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

func strokeRectThickness(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA, thickness int) {
	for i := 0; i < thickness; i++ {
		inner := image.Rect(rect.Min.X+i, rect.Min.Y+i, rect.Max.X-i, rect.Max.Y-i)
		if inner.Dx() <= 0 || inner.Dy() <= 0 {
			return
		}
		strokeRect(img, inner, stroke)
	}
}

func drawCross(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA) {
	margin := max(2, min(rect.Dx(), rect.Dy())/5)
	x1 := rect.Min.X + margin
	y1 := rect.Min.Y + margin
	x2 := rect.Max.X - margin - 1
	y2 := rect.Max.Y - margin - 1
	thickness := max(2, min(rect.Dx(), rect.Dy())/10)
	drawThickLine(img, x1, y1, x2, y2, thickness, stroke)
	drawThickLine(img, x2, y1, x1, y2, thickness, stroke)
}

func crossColor(fill color.NRGBA) color.NRGBA {
	if perceivedBrightness(fill) < 96 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.NRGBA{R: 20, G: 20, B: 20, A: 255}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func drawThickLine(img *image.RGBA, x1, y1, x2, y2, thickness int, stroke color.NRGBA) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	steps := max(absInt(x2-x1), absInt(y2-y1))
	if steps == 0 {
		paintSquare(img, x1, y1, thickness, stroke)
		return
	}
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		x := int(math.Round(float64(x1) + dx*t))
		y := int(math.Round(float64(y1) + dy*t))
		paintSquare(img, x, y, thickness, stroke)
	}
}

func paintSquare(img *image.RGBA, x, y, thickness int, stroke color.NRGBA) {
	radius := thickness / 2
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			px := x + dx
			py := y + dy
			if image.Pt(px, py).In(img.Bounds()) {
				img.Set(px, py, stroke)
			}
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func perceivedBrightness(c color.NRGBA) uint8 {
	value := (299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000
	return uint8(value)
}
