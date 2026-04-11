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

func TestReplacePaletteColorUpdatesSourceForNewHex(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(2, 1)
	color := doc.EnsurePaletteColor("#112233")
	color.Name = "Old name"
	doc.Palette[0] = color
	if err := doc.SetBeadColor(0, 0, color.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}

	replacement, changed, err := doc.ReplacePaletteColor(color.ID, "#445566")
	if err != nil {
		t.Fatalf("ReplacePaletteColor() error = %v", err)
	}
	if !changed {
		t.Fatal("expected replacement to change the document")
	}
	if replacement.ID != color.ID || replacement.Index != color.Index {
		t.Fatalf("expected source color identity to be preserved, got %+v", replacement)
	}
	if replacement.Hex != "#445566" {
		t.Fatalf("expected replacement hex, got %q", replacement.Hex)
	}
	if replacement.Name != "" {
		t.Fatalf("expected stale name to be cleared, got %q", replacement.Name)
	}
	if doc.Beads[0][0].ColorID != color.ID {
		t.Fatalf("expected bead reference to stay on source ID, got %q", doc.Beads[0][0].ColorID)
	}
	if len(doc.Palette) != 1 {
		t.Fatalf("expected palette length to stay 1, got %d", len(doc.Palette))
	}
}

func TestReplacePaletteColorMergesExistingHex(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(3, 1)
	source := doc.EnsurePaletteColor("#112233")
	target := doc.EnsurePaletteColor("#AABBCC")
	_ = doc.SetBeadColor(0, 0, source.ID)
	_ = doc.SetBeadColor(0, 1, target.ID)
	_ = doc.SetBeadColor(0, 2, source.ID)

	replacement, changed, err := doc.ReplacePaletteColor(source.ID, target.Hex)
	if err != nil {
		t.Fatalf("ReplacePaletteColor() error = %v", err)
	}
	if !changed {
		t.Fatal("expected replacement to change the document")
	}
	if replacement.ID != target.ID {
		t.Fatalf("expected target color to remain, got %+v", replacement)
	}
	if len(doc.Palette) != 1 {
		t.Fatalf("expected source color to be removed, got palette %+v", doc.Palette)
	}
	if doc.Palette[0].ID != target.ID || doc.Palette[0].Index != target.Index {
		t.Fatalf("expected target palette entry to be preserved, got %+v", doc.Palette[0])
	}
	for col := range doc.Beads[0] {
		if doc.Beads[0][col].ColorID != target.ID {
			t.Fatalf("expected bead %d to reference target ID, got %q", col, doc.Beads[0][col].ColorID)
		}
	}
}

func TestReplacePaletteColorSameHexNoop(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(1, 1)
	color := doc.EnsurePaletteColor("#112233")

	replacement, changed, err := doc.ReplacePaletteColor(color.ID, "#112233")
	if err != nil {
		t.Fatalf("ReplacePaletteColor() error = %v", err)
	}
	if changed {
		t.Fatal("expected same-color replacement to be a no-op")
	}
	if replacement.ID != color.ID {
		t.Fatalf("expected original color, got %+v", replacement)
	}
	if len(doc.Palette) != 1 {
		t.Fatalf("expected palette to stay unchanged, got %+v", doc.Palette)
	}
}

func TestReplacePaletteColorUnknownSource(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(1, 1)

	if _, _, err := doc.ReplacePaletteColor("missing", "#112233"); err == nil {
		t.Fatal("expected error for unknown source color")
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

func TestResizeShrinksFromBottomAndLeft(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(3, 3)
	color := doc.EnsurePaletteColor("#112233")
	_ = doc.SetBeadColor(0, 0, color.ID)
	_ = doc.SetBeadColor(0, 1, color.ID)
	_ = doc.SetBeadColor(2, 2, color.ID)

	if err := doc.Resize(2, 2); err != nil {
		t.Fatalf("Resize() error = %v", err)
	}

	if doc.Canvas.Width != 2 || doc.Canvas.Height != 2 {
		t.Fatalf("unexpected size after resize: %+v", doc.Canvas)
	}
	if doc.Beads[0][0].ColorID != color.ID {
		t.Fatalf("expected old column 1 to become new column 0, got %q", doc.Beads[0][0].ColorID)
	}
	if len(doc.Beads) != 2 {
		t.Fatalf("expected bottom rows to be removed")
	}
}

func TestResizeGrowsRowsBottomAndColumnsLeft(t *testing.T) {
	t.Parallel()

	doc, _ := NewDocument(2, 2)
	color := doc.EnsurePaletteColor("#445566")
	_ = doc.SetBeadColor(0, 0, color.ID)

	if err := doc.Resize(4, 3); err != nil {
		t.Fatalf("Resize() error = %v", err)
	}

	if doc.Canvas.Width != 4 || doc.Canvas.Height != 3 {
		t.Fatalf("unexpected size after resize: %+v", doc.Canvas)
	}
	if doc.Beads[0][2].ColorID != color.ID {
		t.Fatalf("expected original first column to shift right after left padding, got %q", doc.Beads[0][2].ColorID)
	}
	if len(doc.Beads[2]) != 4 {
		t.Fatalf("expected new bottom row to match new width")
	}
}
