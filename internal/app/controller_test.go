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
	if controller.HasDocument() {
		t.Fatal("new controller should start without an open document")
	}

	if err := controller.NewDocument(4, 4); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
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

	if err := controller.NewDocument(4, 4); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
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

	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
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

func TestControllerResizeDocument(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(3, 3); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	controller.SetSelectionTarget(model.SelectionRow)
	if err := controller.ActivateBead(1, 0); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}

	if err := controller.ResizeDocument(2, 4); err != nil {
		t.Fatalf("ResizeDocument() error = %v", err)
	}

	if controller.Session().Document.Canvas.Width != 2 || controller.Session().Document.Canvas.Height != 4 {
		t.Fatalf("unexpected canvas size after resize: %+v", controller.Session().Document.Canvas)
	}
	if controller.Session().Selection.Mode != model.SelectionNone {
		t.Fatal("expected selection to be cleared after resize")
	}
	if !controller.Session().Dirty {
		t.Fatal("expected resize to mark document dirty")
	}
}

func TestControllerZoomOutSupportsLargeMaps(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}

	for range 20 {
		controller.ZoomOut()
	}

	if controller.Session().Zoom != MinZoom {
		t.Fatalf("expected zoom to clamp at %v, got %v", MinZoom, controller.Session().Zoom)
	}
	if controller.Session().Zoom > 0.1 {
		t.Fatalf("expected zoom to support large-map overview, got %v", controller.Session().Zoom)
	}
}

func TestControllerLoadImportedDocument(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	notifications := 0
	controller.Subscribe(func() {
		notifications++
	})

	doc, err := model.NewDocument(2, 1)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	color := doc.EnsurePaletteColor("#112233")
	if err := doc.SetBeadColor(0, 0, color.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}

	if err := controller.LoadImportedDocument(doc, "sample.png"); err != nil {
		t.Fatalf("LoadImportedDocument() error = %v", err)
	}

	session := controller.Session()
	if session.Document != doc {
		t.Fatal("expected imported document to become active")
	}
	if session.FilePath != "" {
		t.Fatalf("expected imported document to have no file path, got %q", session.FilePath)
	}
	if !session.Dirty {
		t.Fatal("expected imported document to be dirty")
	}
	if session.CurrentTool != model.ToolPaint {
		t.Fatalf("expected paint tool, got %q", session.CurrentTool)
	}
	if session.Selection.Mode != model.SelectionNone || session.SelectionTarget != model.SelectionNone {
		t.Fatalf("expected selection to be reset, got selection=%+v target=%q", session.Selection, session.SelectionTarget)
	}
	if session.Zoom != 1 {
		t.Fatalf("expected zoom to reset to 1, got %v", session.Zoom)
	}
	if session.SelectedColor.ID != color.ID {
		t.Fatalf("expected first palette colour to be selected, got %+v", session.SelectedColor)
	}
	if notifications != 1 {
		t.Fatalf("expected one notification, got %d", notifications)
	}
}
