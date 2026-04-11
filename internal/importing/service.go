package importing

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kostya/peyote-designer/internal/model"
)

const (
	DefaultBeadCount  = 240
	MinBeadCount      = 1
	MaxBeadCount      = 10000
	DefaultColorCount = 120
	MaxColorCount     = 120
	MaxImagePixels    = 50_000_000
)

var (
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrInvalidSelection  = errors.New("invalid image selection")
	ErrInvalidConfig     = errors.New("invalid import configuration")
	ErrImageTooLarge     = errors.New("image is too large")
	ErrDecodeImage       = errors.New("decode image")
)

type Service struct{}

type SourceImage struct {
	Path   string
	Format string
	Image  image.Image
	Bounds image.Rectangle
}

type Config struct {
	BeadCount  int
	ColorCount int
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) LoadImage(path string) (*SourceImage, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !isSupportedExtension(ext) {
		return nil, fmt.Errorf("%w: use a PNG or JPEG image", ErrUnsupportedFormat)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeImage, err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("%w: image has invalid dimensions", ErrDecodeImage)
	}
	if config.Width*config.Height > MaxImagePixels {
		return nil, fmt.Errorf("%w: maximum supported size is %d pixels", ErrImageTooLarge, MaxImagePixels)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("reset image reader: %w", err)
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeImage, err)
	}
	if !isSupportedFormat(format) {
		return nil, fmt.Errorf("%w: use a PNG or JPEG image", ErrUnsupportedFormat)
	}

	return &SourceImage{
		Path:   path,
		Format: format,
		Image:  img,
		Bounds: img.Bounds(),
	}, nil
}

func (s *Service) GridSize(selection image.Rectangle, beadCount int) (int, int, error) {
	if beadCount < MinBeadCount || beadCount > MaxBeadCount {
		return 0, 0, fmt.Errorf("%w: bead count must be between %d and %d",
			ErrInvalidConfig,
			MinBeadCount,
			MaxBeadCount,
		)
	}
	if selection.Dx() <= 0 || selection.Dy() <= 0 {
		return 0, 0, ErrInvalidSelection
	}

	aspect := float64(selection.Dx()) / float64(selection.Dy())
	bestWidth := 1
	bestHeight := beadCount
	bestScore := math.MaxFloat64
	for width := 1; width <= beadCount; width++ {
		height := maxInt(1, int(math.Round(float64(beadCount)/float64(width))))
		total := width * height
		gridAspect := float64(width) / float64(height)
		totalPenalty := float64(absInt(total-beadCount)) * 1000
		aspectPenalty := math.Abs(math.Log(gridAspect / aspect))
		score := totalPenalty + aspectPenalty
		if score < bestScore {
			bestScore = score
			bestWidth = width
			bestHeight = height
		}
	}
	return bestWidth, bestHeight, nil
}

func (s *Service) Convert(source *SourceImage, selection image.Rectangle, config Config) (*model.Document, error) {
	if source == nil || source.Image == nil {
		return nil, fmt.Errorf("%w: no source image", ErrDecodeImage)
	}
	selection = selection.Canon().Intersect(source.Bounds)
	if selection.Dx() <= 0 || selection.Dy() <= 0 {
		return nil, ErrInvalidSelection
	}
	if config.ColorCount < 0 || config.ColorCount > MaxColorCount {
		return nil, fmt.Errorf("%w: colour count must be between 0 and %d", ErrInvalidConfig, MaxColorCount)
	}

	width, height, err := s.GridSize(selection, config.BeadCount)
	if err != nil {
		return nil, err
	}
	samples := sampleGrid(source.Image, selection, width, height)
	palette := reducePalette(samples, effectiveColorCount(samples, config.ColorCount))

	doc, err := model.NewDocument(width, height)
	if err != nil {
		return nil, err
	}
	doc.SetTitle("Imported Pattern")
	doc.Palette = []model.PaletteColor{}
	doc.Beads = make([][]model.Bead, height)
	for row := 0; row < height; row++ {
		doc.Beads[row] = make([]model.Bead, width)
		for col := 0; col < width; col++ {
			nearest := nearestColor(samples[row*width+col], palette)
			paletteColor := doc.EnsurePaletteColor(hexColor(nearest))
			doc.Beads[row][col].ColorID = paletteColor.ID
		}
	}
	doc.View = model.ViewState{
		Zoom:         1,
		SelectedTool: model.ToolPaint,
	}
	return doc, doc.Validate()
}

func FriendlyError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrUnsupportedFormat):
		return "Choose a PNG or JPEG image."
	case errors.Is(err, ErrImageTooLarge):
		return "That image is too large to import. Try cropping or resizing it first."
	case errors.Is(err, ErrDecodeImage):
		return "The image could not be read. Choose a valid PNG or JPEG file."
	case errors.Is(err, ErrInvalidSelection):
		return "Select an area of the image before continuing."
	case errors.Is(err, ErrInvalidConfig):
		return err.Error()
	default:
		return err.Error()
	}
}

func isSupportedExtension(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}

func isSupportedFormat(format string) bool {
	switch strings.ToLower(format) {
	case "png", "jpeg":
		return true
	default:
		return false
	}
}

func sampleGrid(img image.Image, selection image.Rectangle, width, height int) []color.NRGBA {
	out := make([]color.NRGBA, width*height)
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			out[row*width+col] = sampleCell(img, selection, col, row, width, height)
		}
	}
	return out
}

func sampleCell(img image.Image, selection image.Rectangle, col, row, width, height int) color.NRGBA {
	var red, green, blue int
	samples := 0
	for sy := 0; sy < 3; sy++ {
		y := selection.Min.Y + int((float64(row)+float64(sy+1)/4)*float64(selection.Dy())/float64(height))
		y = clampInt(y, selection.Min.Y, selection.Max.Y-1)
		for sx := 0; sx < 3; sx++ {
			x := selection.Min.X + int((float64(col)+float64(sx+1)/4)*float64(selection.Dx())/float64(width))
			x = clampInt(x, selection.Min.X, selection.Max.X-1)
			c := flattenColor(img.At(x, y))
			red += int(c.R)
			green += int(c.G)
			blue += int(c.B)
			samples++
		}
	}
	return color.NRGBA{
		R: uint8(red / samples),
		G: uint8(green / samples),
		B: uint8(blue / samples),
		A: 255,
	}
}

func flattenColor(c color.Color) color.NRGBA {
	red, green, blue, alpha := c.RGBA()
	if alpha == 0 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if alpha == 0xffff {
		return color.NRGBA{
			R: uint8(red >> 8),
			G: uint8(green >> 8),
			B: uint8(blue >> 8),
			A: 255,
		}
	}

	invAlpha := 0xffff - alpha
	return color.NRGBA{
		R: uint8((red + invAlpha) >> 8),
		G: uint8((green + invAlpha) >> 8),
		B: uint8((blue + invAlpha) >> 8),
		A: 255,
	}
}

func effectiveColorCount(samples []color.NRGBA, requested int) int {
	unique := make(map[color.NRGBA]struct{}, len(samples))
	for _, sample := range samples {
		unique[sample] = struct{}{}
	}
	if requested == 0 {
		return minInt(MaxColorCount, maxInt(1, len(unique)))
	}
	return minInt(requested, maxInt(1, len(unique)))
}

func reducePalette(samples []color.NRGBA, limit int) []color.NRGBA {
	unique := weightedSamples(samples)
	if len(unique) <= limit {
		out := make([]color.NRGBA, 0, len(unique))
		for _, sample := range unique {
			out = append(out, sample.Color)
		}
		sortColors(out)
		return out
	}

	boxes := []colorBox{{Samples: unique}}
	for len(boxes) < limit {
		index := splitCandidate(boxes)
		if index < 0 {
			break
		}
		left, right := splitBox(boxes[index])
		boxes = append(boxes[:index], boxes[index+1:]...)
		boxes = append(boxes, left, right)
	}

	out := make([]color.NRGBA, 0, len(boxes))
	for _, box := range boxes {
		out = append(out, averageBox(box))
	}
	sortColors(out)
	return out
}

type weightedColor struct {
	Color color.NRGBA
	Count int
}

type colorBox struct {
	Samples []weightedColor
}

func weightedSamples(samples []color.NRGBA) []weightedColor {
	counts := make(map[color.NRGBA]int, len(samples))
	for _, sample := range samples {
		counts[sample]++
	}
	out := make([]weightedColor, 0, len(counts))
	for sample, count := range counts {
		out = append(out, weightedColor{Color: sample, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		return colorLess(out[i].Color, out[j].Color)
	})
	return out
}

func splitCandidate(boxes []colorBox) int {
	bestIndex := -1
	bestScore := -1
	for index, box := range boxes {
		if len(box.Samples) < 2 {
			continue
		}
		redRange, greenRange, blueRange := channelRanges(box.Samples)
		score := maxInt(redRange, maxInt(greenRange, blueRange)) * totalWeight(box.Samples)
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	return bestIndex
}

func splitBox(box colorBox) (colorBox, colorBox) {
	redRange, greenRange, blueRange := channelRanges(box.Samples)
	channel := 0
	if greenRange >= redRange && greenRange >= blueRange {
		channel = 1
	}
	if blueRange >= redRange && blueRange >= greenRange {
		channel = 2
	}

	samples := append([]weightedColor(nil), box.Samples...)
	sort.Slice(samples, func(i, j int) bool {
		left := channelValue(samples[i].Color, channel)
		right := channelValue(samples[j].Color, channel)
		if left == right {
			return colorLess(samples[i].Color, samples[j].Color)
		}
		return left < right
	})

	halfWeight := totalWeight(samples) / 2
	running := 0
	splitAt := 1
	for index, sample := range samples {
		running += sample.Count
		if running >= halfWeight {
			splitAt = clampInt(index+1, 1, len(samples)-1)
			break
		}
	}
	return colorBox{Samples: samples[:splitAt]}, colorBox{Samples: samples[splitAt:]}
}

func channelRanges(samples []weightedColor) (int, int, int) {
	minR, minG, minB := 255, 255, 255
	maxR, maxG, maxB := 0, 0, 0
	for _, sample := range samples {
		c := sample.Color
		minR = minInt(minR, int(c.R))
		minG = minInt(minG, int(c.G))
		minB = minInt(minB, int(c.B))
		maxR = maxInt(maxR, int(c.R))
		maxG = maxInt(maxG, int(c.G))
		maxB = maxInt(maxB, int(c.B))
	}
	return maxR - minR, maxG - minG, maxB - minB
}

func averageBox(box colorBox) color.NRGBA {
	var red, green, blue, total int
	for _, sample := range box.Samples {
		red += int(sample.Color.R) * sample.Count
		green += int(sample.Color.G) * sample.Count
		blue += int(sample.Color.B) * sample.Count
		total += sample.Count
	}
	if total == 0 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.NRGBA{
		R: uint8(red / total),
		G: uint8(green / total),
		B: uint8(blue / total),
		A: 255,
	}
}

func nearestColor(sample color.NRGBA, palette []color.NRGBA) color.NRGBA {
	if len(palette) == 0 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	best := palette[0]
	bestDistance := math.MaxInt
	for _, candidate := range palette {
		distance := colorDistance(sample, candidate)
		if distance < bestDistance || distance == bestDistance && colorLess(candidate, best) {
			best = candidate
			bestDistance = distance
		}
	}
	return best
}

func colorDistance(left, right color.NRGBA) int {
	red := int(left.R) - int(right.R)
	green := int(left.G) - int(right.G)
	blue := int(left.B) - int(right.B)
	return red*red + green*green + blue*blue
}

func channelValue(c color.NRGBA, channel int) int {
	switch channel {
	case 0:
		return int(c.R)
	case 1:
		return int(c.G)
	default:
		return int(c.B)
	}
}

func totalWeight(samples []weightedColor) int {
	total := 0
	for _, sample := range samples {
		total += sample.Count
	}
	return total
}

func hexColor(c color.NRGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func sortColors(colors []color.NRGBA) {
	sort.Slice(colors, func(i, j int) bool {
		return colorLess(colors[i], colors[j])
	})
}

func colorLess(left, right color.NRGBA) bool {
	if left.R != right.R {
		return left.R < right.R
	}
	if left.G != right.G {
		return left.G < right.G
	}
	return left.B < right.B
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
