package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kostya/peyote-designer/internal/model"
)

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewStore()
	doc, _ := model.NewDocument(2, 2)
	color := doc.EnsurePaletteColor("#ABCDEF")
	if err := doc.SetBeadColor(0, 1, color.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}
	if err := doc.ToggleCompleted(0, 1); err != nil {
		t.Fatalf("ToggleCompleted() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "sample.pey")
	if err := store.Save(path, doc); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Beads[0][1].ColorID != color.ID {
		t.Fatalf("expected color %q, got %q", color.ID, loaded.Beads[0][1].ColorID)
	}
	if !loaded.Beads[0][1].Completed {
		t.Fatal("expected completed bead to persist")
	}
}

func TestStoreAppendsExtension(t *testing.T) {
	t.Parallel()

	store := NewStore()
	doc, _ := model.NewDocument(1, 1)
	path := filepath.Join(t.TempDir(), "missing-extension")

	if err := store.Save(path, doc); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(path + ".pey"); err != nil {
		t.Fatalf("expected saved file with .pey extension: %v", err)
	}
}
