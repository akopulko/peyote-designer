package app

import (
	"log/slog"
	"testing"

	"github.com/kostya/peyote-designer/internal/model"
	"github.com/kostya/peyote-designer/internal/persistence"
)

func TestControllerDirtyState(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}

	if controller.Session().Dirty {
		t.Fatal("new controller should start clean")
	}

	if err := controller.ActivateBead(0, 0); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}
	if !controller.Session().Dirty {
		t.Fatal("expected dirty state after edit")
	}
}

func TestControllerSelectionAndRemoval(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}

	controller.SetSelectionTarget(model.SelectionRow)
	if err := controller.ActivateBead(1, 0); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}
	if !controller.CanRemoveRow() {
		t.Fatal("expected row removal to be available")
	}
	if err := controller.RemoveSelectedRow(); err != nil {
		t.Fatalf("RemoveSelectedRow() error = %v", err)
	}

	controller.SetSelectionTarget(model.SelectionColumn)
	if err := controller.ActivateBead(0, 1); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}
	if !controller.CanRemoveColumn() {
		t.Fatal("expected column removal to be available")
	}
}

func TestControllerSaveLifecycle(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}

	controller.SetSelectedColor("#123456")
	path := t.TempDir() + "/test-file"
	if err := controller.SaveAs(path); err != nil {
		t.Fatalf("SaveAs() error = %v", err)
	}
	if controller.Session().Dirty {
		t.Fatal("expected clean state after save")
	}
}
