package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AppName       = "Peyote Designer"
	SchemaVersion = 1
)

type Tool string

const (
	ToolPaint  Tool = "paint"
	ToolColor  Tool = "color"
	ToolEraser Tool = "eraser"
	ToolMark   Tool = "mark"
)

type SelectionMode string

const (
	SelectionNone   SelectionMode = "none"
	SelectionRow    SelectionMode = "row"
	SelectionColumn SelectionMode = "column"
)

type Selection struct {
	Mode  SelectionMode `json:"mode"`
	Index int           `json:"index"`
}

type Metadata struct {
	AppName   string    `json:"appName"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Canvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type PaletteColor struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
	Name  string `json:"name,omitempty"`
	Hex   string `json:"hex"`
}

type Bead struct {
	ColorID   string `json:"colorId,omitempty"`
	Completed bool   `json:"completed"`
}

type ViewState struct {
	Zoom            float32 `json:"zoom,omitempty"`
	SelectedTool    Tool    `json:"selectedTool,omitempty"`
	SelectedColorID string  `json:"selectedColorId,omitempty"`
}

type Document struct {
	Version    int                        `json:"version"`
	Metadata   Metadata                   `json:"metadata"`
	Canvas     Canvas                     `json:"canvas"`
	Palette    []PaletteColor             `json:"palette"`
	Beads      [][]Bead                   `json:"beads"`
	View       ViewState                  `json:"view,omitempty"`
	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
}

type Stats struct {
	Total      int
	Completed  int
	Incomplete int
}

type PaletteUsage struct {
	Color PaletteColor
	Count int
}

func NewDocument(width, height int) (*Document, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("width and height must be greater than zero")
	}

	now := time.Now().UTC()
	beads := make([][]Bead, height)
	for row := 0; row < height; row++ {
		beads[row] = make([]Bead, width)
	}

	return &Document{
		Version: SchemaVersion,
		Metadata: Metadata{
			AppName:   AppName,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Canvas: Canvas{
			Width:  width,
			Height: height,
		},
		Palette:    []PaletteColor{},
		Beads:      beads,
		Extensions: map[string]json.RawMessage{},
		View: ViewState{
			Zoom:         1,
			SelectedTool: ToolPaint,
		},
	}, nil
}

func (d *Document) Validate() error {
	if d.Version == 0 {
		return errors.New("missing schema version")
	}
	if d.Canvas.Width <= 0 || d.Canvas.Height <= 0 {
		return errors.New("invalid canvas dimensions")
	}
	if len(d.Beads) != d.Canvas.Height {
		return fmt.Errorf("expected %d rows, got %d", d.Canvas.Height, len(d.Beads))
	}
	for row := range d.Beads {
		if len(d.Beads[row]) != d.Canvas.Width {
			return fmt.Errorf("expected row %d to have %d columns, got %d", row, d.Canvas.Width, len(d.Beads[row]))
		}
	}
	return nil
}

func (d *Document) Touch() {
	d.Metadata.UpdatedAt = time.Now().UTC()
}

func (d *Document) SetTitle(title string) {
	d.Metadata.Title = strings.TrimSpace(title)
	d.Touch()
}

func (d *Document) Stats() Stats {
	total := d.Canvas.Width * d.Canvas.Height
	completed := 0
	for row := range d.Beads {
		for col := range d.Beads[row] {
			if d.Beads[row][col].Completed {
				completed++
			}
		}
	}
	return Stats{
		Total:      total,
		Completed:  completed,
		Incomplete: total - completed,
	}
}

func (d *Document) PaletteUsage() []PaletteUsage {
	counts := make(map[string]int)
	paletteByID := make(map[string]PaletteColor, len(d.Palette))
	for _, color := range d.Palette {
		paletteByID[color.ID] = color
	}
	for row := range d.Beads {
		for col := range d.Beads[row] {
			if d.Beads[row][col].ColorID == "" {
				continue
			}
			counts[d.Beads[row][col].ColorID]++
		}
	}

	out := make([]PaletteUsage, 0, len(counts))
	for colorID, count := range counts {
		if color, ok := paletteByID[colorID]; ok {
			out = append(out, PaletteUsage{Color: color, Count: count})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Color.Index < out[j].Color.Index
	})
	return out
}

func (d *Document) EnsurePaletteColor(hex string) PaletteColor {
	hex = NormalizeHex(hex)
	for _, color := range d.Palette {
		if color.Hex == hex {
			return color
		}
	}

	color := PaletteColor{
		ID:    fmt.Sprintf("color-%d", len(d.Palette)+1),
		Index: len(d.Palette) + 1,
		Hex:   hex,
	}
	d.Palette = append(d.Palette, color)
	return color
}

func (d *Document) PaletteColorByID(id string) (PaletteColor, bool) {
	for _, color := range d.Palette {
		if color.ID == id {
			return color, true
		}
	}
	return PaletteColor{}, false
}

func (d *Document) SetBeadColor(row, col int, colorID string) error {
	if err := d.validateCoords(row, col); err != nil {
		return err
	}
	d.Beads[row][col].ColorID = colorID
	d.Touch()
	return nil
}

func (d *Document) ClearBead(row, col int) error {
	if err := d.validateCoords(row, col); err != nil {
		return err
	}
	d.Beads[row][col].ColorID = ""
	d.Touch()
	return nil
}

func (d *Document) ToggleCompleted(row, col int) error {
	if err := d.validateCoords(row, col); err != nil {
		return err
	}
	d.Beads[row][col].Completed = !d.Beads[row][col].Completed
	d.Touch()
	return nil
}

func (d *Document) RemoveRow(index int) error {
	if index < 0 || index >= d.Canvas.Height {
		return errors.New("row index out of range")
	}
	d.Beads = append(d.Beads[:index], d.Beads[index+1:]...)
	d.Canvas.Height--
	d.Touch()
	return nil
}

func (d *Document) RemoveColumn(index int) error {
	if index < 0 || index >= d.Canvas.Width {
		return errors.New("column index out of range")
	}
	for row := range d.Beads {
		d.Beads[row] = append(d.Beads[row][:index], d.Beads[row][index+1:]...)
	}
	d.Canvas.Width--
	d.Touch()
	return nil
}

func (d *Document) validateCoords(row, col int) error {
	if row < 0 || row >= d.Canvas.Height {
		return errors.New("row out of range")
	}
	if col < 0 || col >= d.Canvas.Width {
		return errors.New("column out of range")
	}
	return nil
}

func NormalizeHex(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "#000000"
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	return value
}
