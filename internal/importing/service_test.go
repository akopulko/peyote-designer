package importing

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImageSupportsPNGAndJPEG(t *testing.T) {
	t.Parallel()

	service := NewService()
	tests := []struct {
		name     string
		fileName string
		encode   func(string, image.Image, *testing.T)
		format   string
	}{
		{
			name:     "png",
			fileName: "sample.png",
			encode:   writePNG,
			format:   "png",
		},
		{
			name:     "jpeg",
			fileName: "sample.jpg",
			encode:   writeJPEG,
			format:   "jpeg",
		},
		{
			name:     "jpeg extension",
			fileName: "sample.jpeg",
			encode:   writeJPEG,
			format:   "jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), tt.fileName)
			tt.encode(path, testImage(4, 2), t)

			source, err := service.LoadImage(path)
			if err != nil {
				t.Fatalf("LoadImage() error = %v", err)
			}
			if source.Format != tt.format {
				t.Fatalf("expected format %q, got %q", tt.format, source.Format)
			}
			if source.Bounds.Dx() != 4 || source.Bounds.Dy() != 2 {
				t.Fatalf("unexpected bounds: %v", source.Bounds)
			}
		})
	}
}

func TestLoadImageRejectsUnsupportedAndCorruptFiles(t *testing.T) {
	t.Parallel()

	service := NewService()
	dir := t.TempDir()
	unsupportedPath := filepath.Join(dir, "sample.webp")
	if err := os.WriteFile(unsupportedPath, []byte("not an image"), 0o644); err != nil {
		t.Fatalf("write unsupported fixture: %v", err)
	}
	if _, err := service.LoadImage(unsupportedPath); err == nil {
		t.Fatal("expected unsupported format error")
	}

	corruptPath := filepath.Join(dir, "sample.png")
	if err := os.WriteFile(corruptPath, []byte("not an image"), 0o644); err != nil {
		t.Fatalf("write corrupt fixture: %v", err)
	}
	if _, err := service.LoadImage(corruptPath); err == nil {
		t.Fatal("expected corrupt image error")
	}
}

func TestGridSize(t *testing.T) {
	t.Parallel()

	service := NewService()
	tests := []struct {
		name       string
		selection  image.Rectangle
		beadCount  int
		wantWidth  int
		wantHeight int
		wantErr    bool
	}{
		{
			name:       "landscape exact",
			selection:  image.Rect(0, 0, 4, 2),
			beadCount:  8,
			wantWidth:  4,
			wantHeight: 2,
		},
		{
			name:      "invalid selection",
			selection: image.Rect(0, 0, 0, 2),
			beadCount: 8,
			wantErr:   true,
		},
		{
			name:      "bead count too low",
			selection: image.Rect(0, 0, 4, 2),
			beadCount: 0,
			wantErr:   true,
		},
		{
			name:      "bead count too high",
			selection: image.Rect(0, 0, 4, 2),
			beadCount: MaxBeadCount + 1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			width, height, err := service.GridSize(tt.selection, tt.beadCount)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GridSize() error = %v", err)
			}
			if width != tt.wantWidth || height != tt.wantHeight {
				t.Fatalf("expected %dx%d, got %dx%d", tt.wantWidth, tt.wantHeight, width, height)
			}
		})
	}
}

func TestConvertProducesValidPaletteLimitedDocument(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := &SourceImage{
		Path:   "fixture.png",
		Format: "png",
		Image:  testImage(4, 2),
		Bounds: image.Rect(0, 0, 4, 2),
	}

	doc, err := service.Convert(source, source.Bounds, Config{BeadCount: 8, ColorCount: 2})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("imported document did not validate: %v", err)
	}
	if doc.Canvas.Width != 4 || doc.Canvas.Height != 2 {
		t.Fatalf("unexpected canvas size: %+v", doc.Canvas)
	}
	if len(doc.Palette) > 2 {
		t.Fatalf("expected palette to be capped at 2 colours, got %d", len(doc.Palette))
	}
}

func TestConvertAutoColorCountUsesUniqueSampleCount(t *testing.T) {
	t.Parallel()

	service := NewService()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{B: 255, A: 255})
	source := &SourceImage{
		Path:   "fixture.png",
		Format: "png",
		Image:  img,
		Bounds: img.Bounds(),
	}

	doc, err := service.Convert(source, source.Bounds, Config{BeadCount: 3, ColorCount: 0})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(doc.Palette) != 3 {
		t.Fatalf("expected auto colour count to keep 3 unique colours, got %d", len(doc.Palette))
	}
}

func TestConvertIsDeterministic(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := &SourceImage{
		Path:   "fixture.png",
		Format: "png",
		Image:  testImage(4, 2),
		Bounds: image.Rect(0, 0, 4, 2),
	}
	config := Config{BeadCount: 8, ColorCount: 3}

	first, err := service.Convert(source, source.Bounds, config)
	if err != nil {
		t.Fatalf("first Convert() error = %v", err)
	}
	second, err := service.Convert(source, source.Bounds, config)
	if err != nil {
		t.Fatalf("second Convert() error = %v", err)
	}
	if len(first.Palette) != len(second.Palette) {
		t.Fatalf("palette length mismatch: %d != %d", len(first.Palette), len(second.Palette))
	}
	for index := range first.Palette {
		if first.Palette[index].Hex != second.Palette[index].Hex {
			t.Fatalf("palette mismatch at %d: %q != %q", index, first.Palette[index].Hex, second.Palette[index].Hex)
		}
	}
	for row := range first.Beads {
		for col := range first.Beads[row] {
			if first.Beads[row][col].ColorID != second.Beads[row][col].ColorID {
				t.Fatalf("bead mismatch at row=%d col=%d", row, col)
			}
		}
	}
}

func TestConvertRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := &SourceImage{
		Path:   "fixture.png",
		Format: "png",
		Image:  testImage(2, 2),
		Bounds: image.Rect(0, 0, 2, 2),
	}

	tests := []struct {
		name      string
		source    *SourceImage
		selection image.Rectangle
		config    Config
	}{
		{
			name:      "nil source",
			source:    nil,
			selection: image.Rect(0, 0, 1, 1),
			config:    Config{BeadCount: 1, ColorCount: 1},
		},
		{
			name:      "empty selection",
			source:    source,
			selection: image.Rect(0, 0, 0, 1),
			config:    Config{BeadCount: 1, ColorCount: 1},
		},
		{
			name:      "invalid colour count",
			source:    source,
			selection: source.Bounds,
			config:    Config{BeadCount: 4, ColorCount: MaxColorCount + 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := service.Convert(tt.source, tt.selection, tt.config); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func testImage(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	colors := []color.NRGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
		{R: 255, G: 255, A: 255},
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, colors[(x+y)%len(colors)])
		}
	}
	return img
}

func writePNG(path string, img image.Image, t *testing.T) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG fixture: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode PNG fixture: %v", err)
	}
}

func writeJPEG(path string, img image.Image, t *testing.T) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create JPEG fixture: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()
	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode JPEG fixture: %v", err)
	}
}
