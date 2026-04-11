package render

import (
	"log/slog"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	application "github.com/kostya/peyote-designer/internal/app"
	"github.com/kostya/peyote-designer/internal/persistence"
)

func TestComputeMetrics(t *testing.T) {
	t.Parallel()

	metrics := ComputeMetrics(1.5)
	if metrics.BeadWidth <= 16 {
		t.Fatalf("expected bead width to scale, got %d", metrics.BeadWidth)
	}
	expectedHeight := int(math.Round(float64(metrics.BeadWidth) * 1.5))
	if metrics.BeadHeight != expectedHeight {
		t.Fatalf("expected bead height to keep 1:1.5 ratio, got %+v", metrics)
	}
	if metrics.Gap != 3 {
		t.Fatalf("expected gap to scale from reduced base spacing, got %d", metrics.Gap)
	}
}

func TestPeyoteMapSizeIncludesOddRowOffset(t *testing.T) {
	t.Parallel()

	metrics := ComputeMetrics(1)
	size := PeyoteMapSize(3, 3, metrics)

	if size.X != 65 {
		t.Fatalf("expected width to include odd-row half stride, got %d", size.X)
	}
	if size.Y != 80 {
		t.Fatalf("expected height to match row stride, got %d", size.Y)
	}
}

func TestHitTestRejectsGapAndMapsBead(t *testing.T) {
	t.Parallel()

	controller, err := application.NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(3, 3); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	beadMap := NewBeadMap(controller)
	row, col, ok := beadMap.HitTest(fyne.NewPos(10, 10))
	if !ok || row != 0 || col != 0 {
		t.Fatalf("expected first bead hit, got row=%d col=%d ok=%v", row, col, ok)
	}

	_, _, ok = beadMap.HitTest(fyne.NewPos(18, 10))
	if ok {
		t.Fatal("expected gap click to be rejected")
	}
}

func TestHitTestMapsStaggeredOddRows(t *testing.T) {
	t.Parallel()

	controller, err := application.NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(3, 3); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	beadMap := NewBeadMap(controller)
	row, col, ok := beadMap.HitTest(fyne.NewPos(21, 42))
	if !ok || row != 1 || col != 0 {
		t.Fatalf("expected shifted odd-row bead hit, got row=%d col=%d ok=%v", row, col, ok)
	}

	_, _, ok = beadMap.HitTest(fyne.NewPos(2, 42))
	if ok {
		t.Fatal("expected unshifted odd-row position to be rejected")
	}
}
