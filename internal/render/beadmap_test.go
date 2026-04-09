package render

import "testing"

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
