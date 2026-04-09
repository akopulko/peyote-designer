package render

import (
	"log/slog"
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
	if metrics.BeadHeight <= metrics.BeadWidth {
		t.Fatalf("expected bead height to remain larger than width, got %+v", metrics)
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

	_, _, ok = beadMap.HitTest(fyne.NewPos(22, 10))
	if ok {
		t.Fatal("expected gap click to be rejected")
	}
}
