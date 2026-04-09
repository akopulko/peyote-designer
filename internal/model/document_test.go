package model

import (
	"encoding/json"
	"testing"
)

func TestNewDocument(t *testing.T) {
	t.Parallel()

	doc, err := NewDocument(4, 3)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	if doc.Canvas.Width != 4 || doc.Canvas.Height != 3 {
		t.Fatalf("unexpected canvas dimensions: %+v", doc.Canvas)
	}
	if got := len(doc.Beads); got != 3 {
		t.Fatalf("expected 3 rows, got %d", got)
	}
}

func TestPaletteUsageAndStats(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(2, 2)
	color := doc.EnsurePaletteColor("#ff0000")
	if err := doc.SetBeadColor(0, 0, color.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}
	if err := doc.SetBeadColor(1, 1, color.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}
	if err := doc.ToggleCompleted(0, 0); err != nil {
		t.Fatalf("ToggleCompleted() error = %v", err)
	}

	stats := doc.Stats()
	if stats.Total != 4 || stats.Completed != 1 || stats.Incomplete != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	usage := doc.PaletteUsage()
	if len(usage) != 1 || usage[0].Count != 2 {
		t.Fatalf("unexpected palette usage: %+v", usage)
	}
}

func TestRemoveRowAndColumn(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(3, 3)
	if err := doc.RemoveRow(1); err != nil {
		t.Fatalf("RemoveRow() error = %v", err)
	}
	if doc.Canvas.Height != 2 {
		t.Fatalf("expected height 2, got %d", doc.Canvas.Height)
	}
	if err := doc.RemoveColumn(0); err != nil {
		t.Fatalf("RemoveColumn() error = %v", err)
	}
	if doc.Canvas.Width != 2 {
		t.Fatalf("expected width 2, got %d", doc.Canvas.Width)
	}
}

func TestExtensionsRoundTripCompatibility(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(1, 1)
	doc.Extensions["future"] = json.RawMessage(`{"enabled":true}`)

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded Document
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if string(decoded.Extensions["future"]) != `{"enabled":true}` {
		t.Fatalf("unexpected extension payload: %s", decoded.Extensions["future"])
	}
}
