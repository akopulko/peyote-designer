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

func TestControllerSetSelectedColorDoesNotCreatePaletteEntry(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	initialPaletteLength := len(controller.Session().Document.Palette)
	controller.Session().Dirty = false

	controller.SetSelectedColor("#123456")

	if controller.Session().SelectedColor.ID != "" {
		t.Fatalf("expected unpainted selected color to have no palette ID, got %+v", controller.Session().SelectedColor)
	}
	if len(controller.Session().Document.Palette) != initialPaletteLength {
		t.Fatalf("expected palette length %d, got %d", initialPaletteLength, len(controller.Session().Document.Palette))
	}
	if controller.Session().Dirty {
		t.Fatal("expected selecting a color to leave document clean")
	}
}

func TestControllerPaintCreatesSelectedPaletteEntry(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	controller.SetSelectedColor("#123456")

	if err := controller.ActivateBead(0, 0); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}

	session := controller.Session()
	if session.SelectedColor.ID == "" {
		t.Fatal("expected painting to create a palette-backed selected color")
	}
	if session.Document.Beads[0][0].ColorID != session.SelectedColor.ID {
		t.Fatalf("expected bead to use selected color ID, got %q", session.Document.Beads[0][0].ColorID)
	}
	if session.SelectedColor.Hex != "#123456" {
		t.Fatalf("expected selected color hex to be normalized, got %q", session.SelectedColor.Hex)
	}
	if !session.SelectedBead.Active || session.SelectedBead.Row != 0 || session.SelectedBead.Col != 0 {
		t.Fatalf("expected painted bead to be highlighted, got %+v", session.SelectedBead)
	}
	if session.SelectedPaletteColorID != "" {
		t.Fatalf("expected paint action to clear palette highlight, got %q", session.SelectedPaletteColorID)
	}
}

func TestControllerMarkAndEraserHighlightLastActionedBead(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(3, 3); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	controller.SetTool(model.ToolMark)
	if err := controller.ActivateBead(1, 2); err != nil {
		t.Fatalf("ActivateBead() mark error = %v", err)
	}
	if !controller.Session().SelectedBead.Active ||
		controller.Session().SelectedBead.Row != 1 ||
		controller.Session().SelectedBead.Col != 2 {
		t.Fatalf("expected marked bead to be highlighted, got %+v", controller.Session().SelectedBead)
	}

	controller.SetTool(model.ToolEraser)
	if err := controller.ActivateBead(2, 1); err != nil {
		t.Fatalf("ActivateBead() eraser error = %v", err)
	}
	if !controller.Session().SelectedBead.Active ||
		controller.Session().SelectedBead.Row != 2 ||
		controller.Session().SelectedBead.Col != 1 {
		t.Fatalf("expected erased bead to be highlighted, got %+v", controller.Session().SelectedBead)
	}
	if controller.Session().SelectedPaletteColorID != "" {
		t.Fatalf("expected eraser action to clear palette highlight, got %q", controller.Session().SelectedPaletteColorID)
	}
}

func TestControllerSelectToolSelectsColoredBeadWithoutChangingActiveColor(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	doc := controller.Session().Document
	beadColor := doc.EnsurePaletteColor("#123456")
	if err := doc.SetBeadColor(1, 1, beadColor.ID); err != nil {
		t.Fatalf("SetBeadColor() error = %v", err)
	}
	controller.SetSelectedColor("#ABCDEF")
	activeColor := controller.Session().SelectedColor
	controller.Session().Dirty = false

	controller.SetTool(model.ToolSelect)
	if err := controller.ActivateBead(1, 1); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}

	session := controller.Session()
	if !session.SelectedBead.Active || session.SelectedBead.Row != 1 || session.SelectedBead.Col != 1 {
		t.Fatalf("expected bead 1,1 to be selected, got %+v", session.SelectedBead)
	}
	if session.SelectedPaletteColorID != beadColor.ID {
		t.Fatalf("expected selected palette color %q, got %q", beadColor.ID, session.SelectedPaletteColorID)
	}
	if session.SelectedColor != activeColor {
		t.Fatalf("expected active color to stay %+v, got %+v", activeColor, session.SelectedColor)
	}
	if session.Dirty {
		t.Fatal("expected selecting a bead to leave document clean")
	}
}

func TestControllerSelectToolSelectsEmptyBeadWithoutPaletteHighlight(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	controller.Session().Dirty = false
	controller.SetTool(model.ToolSelect)

	if err := controller.ActivateBead(0, 1); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}

	session := controller.Session()
	if !session.SelectedBead.Active || session.SelectedBead.Row != 0 || session.SelectedBead.Col != 1 {
		t.Fatalf("expected bead 0,1 to be selected, got %+v", session.SelectedBead)
	}
	if session.SelectedPaletteColorID != "" {
		t.Fatalf("expected no palette highlight for empty bead, got %q", session.SelectedPaletteColorID)
	}
	if session.Dirty {
		t.Fatal("expected selecting an empty bead to leave document clean")
	}
}

func TestControllerSetToolClearsSelectedBead(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 2); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	controller.SetTool(model.ToolSelect)
	if err := controller.ActivateBead(0, 0); err != nil {
		t.Fatalf("ActivateBead() error = %v", err)
	}

	controller.SetTool(model.ToolPaint)

	if controller.Session().SelectedBead.Active {
		t.Fatalf("expected selected bead to clear, got %+v", controller.Session().SelectedBead)
	}
	if controller.Session().SelectedPaletteColorID != "" {
		t.Fatalf("expected palette highlight to clear, got %q", controller.Session().SelectedPaletteColorID)
	}
}

func TestControllerReplacePaletteColor(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(2, 1); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	doc := controller.Session().Document
	source := doc.EnsurePaletteColor("#112233")
	target := doc.EnsurePaletteColor("#445566")
	_ = doc.SetBeadColor(0, 0, source.ID)
	_ = doc.SetBeadColor(0, 1, target.ID)
	controller.Session().Dirty = false

	if err := controller.ReplacePaletteColor(source.ID, target.Hex); err != nil {
		t.Fatalf("ReplacePaletteColor() error = %v", err)
	}

	session := controller.Session()
	if !session.Dirty {
		t.Fatal("expected replacement to mark document dirty")
	}
	if session.SelectedColor.ID != target.ID {
		t.Fatalf("expected selected color to become target palette color, got %+v", session.SelectedColor)
	}
	for col := range session.Document.Beads[0] {
		if session.Document.Beads[0][col].ColorID != target.ID {
			t.Fatalf("expected bead %d to reference target ID, got %q", col, session.Document.Beads[0][col].ColorID)
		}
	}
}

func TestControllerReplacePaletteColorSameHexNoop(t *testing.T) {
	t.Parallel()

	controller, err := NewController(persistence.NewStore(), slog.Default())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if err := controller.NewDocument(1, 1); err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}
	source := controller.Session().Document.EnsurePaletteColor("#112233")
	paletteLength := len(controller.Session().Document.Palette)
	controller.Session().Dirty = false

	if err := controller.ReplacePaletteColor(source.ID, source.Hex); err != nil {
		t.Fatalf("ReplacePaletteColor() error = %v", err)
	}

	if controller.Session().Dirty {
		t.Fatal("expected same-color replacement to leave document clean")
	}
	if len(controller.Session().Document.Palette) != paletteLength {
		t.Fatalf("expected palette to stay unchanged, got %+v", controller.Session().Document.Palette)
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
