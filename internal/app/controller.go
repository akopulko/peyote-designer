package app

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/kostya/peyote-designer/internal/model"
	"github.com/kostya/peyote-designer/internal/persistence"
)

const (
	MinZoom = 0.5
	MaxZoom = 3.0
)

type Session struct {
	Document        *model.Document
	FilePath        string
	Dirty           bool
	CurrentTool     model.Tool
	SelectionTarget model.SelectionMode
	Selection       model.Selection
	SelectedColor   model.PaletteColor
	Zoom            float32
}

type Controller struct {
	store       *persistence.Store
	logger      *slog.Logger
	session     *Session
	subscribers []func()
}

func NewController(store *persistence.Store, logger *slog.Logger) (*Controller, error) {
	doc, err := model.NewDocument(12, 20)
	if err != nil {
		return nil, err
	}
	defaultColor := doc.EnsurePaletteColor("#000000")
	session := &Session{
		Document:        doc,
		CurrentTool:     model.ToolPaint,
		SelectionTarget: model.SelectionNone,
		Selection:       model.Selection{Mode: model.SelectionNone, Index: -1},
		SelectedColor:   defaultColor,
		Zoom:            1,
	}
	return &Controller{store: store, logger: logger, session: session}, nil
}

func (c *Controller) Session() *Session {
	return c.session
}

func (c *Controller) Subscribe(fn func()) {
	c.subscribers = append(c.subscribers, fn)
}

func (c *Controller) notify() {
	for _, fn := range c.subscribers {
		fn()
	}
}

func (c *Controller) NewDocument(width, height int) error {
	doc, err := model.NewDocument(width, height)
	if err != nil {
		return err
	}
	color := doc.EnsurePaletteColor(c.session.SelectedColor.Hex)
	c.session = &Session{
		Document:        doc,
		CurrentTool:     model.ToolPaint,
		SelectionTarget: model.SelectionNone,
		Selection:       model.Selection{Mode: model.SelectionNone, Index: -1},
		SelectedColor:   color,
		Zoom:            1,
	}
	c.logger.Info("new document created", "width", width, "height", height)
	c.notify()
	return nil
}

func (c *Controller) LoadDocument(path string) error {
	doc, err := c.store.Load(path)
	if err != nil {
		return err
	}
	selectedColor := model.PaletteColor{ID: "", Index: 0, Hex: "#000000"}
	if len(doc.Palette) > 0 {
		selectedColor = doc.Palette[0]
	}
	c.session = &Session{
		Document:        doc,
		FilePath:        path,
		Dirty:           false,
		CurrentTool:     model.ToolPaint,
		SelectionTarget: model.SelectionNone,
		Selection:       model.Selection{Mode: model.SelectionNone, Index: -1},
		SelectedColor:   selectedColor,
		Zoom:            chooseZoom(doc.View.Zoom),
	}
	c.logger.Info("document loaded", "path", path)
	c.notify()
	return nil
}

func (c *Controller) Save() error {
	if c.session.FilePath == "" {
		return errors.New("document has no file path")
	}
	return c.SaveAs(c.session.FilePath)
}

func (c *Controller) SaveAs(path string) error {
	if c.session.Document == nil {
		return errors.New("no active document")
	}
	c.session.Document.View = model.ViewState{
		Zoom:            c.session.Zoom,
		SelectedTool:    c.session.CurrentTool,
		SelectedColorID: c.session.SelectedColor.ID,
	}
	if err := c.store.Save(path, c.session.Document); err != nil {
		c.logger.Error("save failed", "path", path, "error", err)
		return err
	}
	c.session.FilePath = withPEYExtension(path)
	c.session.Dirty = false
	c.logger.Info("document saved", "path", c.session.FilePath)
	c.notify()
	return nil
}

func (c *Controller) SetTool(tool model.Tool) {
	c.session.CurrentTool = tool
	c.session.SelectionTarget = model.SelectionNone
	c.logger.Info("tool selected", "tool", tool)
	c.notify()
}

func (c *Controller) SetSelectionTarget(mode model.SelectionMode) {
	c.session.SelectionTarget = mode
	c.logger.Info("selection target changed", "mode", mode)
	c.notify()
}

func (c *Controller) SetSelectedColor(hex string) {
	color := c.session.Document.EnsurePaletteColor(hex)
	c.session.SelectedColor = color
	c.session.Dirty = true
	c.logger.Info("selected colour changed", "color", color.Hex)
	c.notify()
}

func (c *Controller) ActivateBead(row, col int) error {
	if c.session.SelectionTarget == model.SelectionRow {
		c.session.Selection = model.Selection{Mode: model.SelectionRow, Index: row}
		c.logger.Info("row selected", "row", row)
		c.notify()
		return nil
	}
	if c.session.SelectionTarget == model.SelectionColumn {
		c.session.Selection = model.Selection{Mode: model.SelectionColumn, Index: col}
		c.logger.Info("column selected", "column", col)
		c.notify()
		return nil
	}

	var err error
	switch c.session.CurrentTool {
	case model.ToolPaint:
		err = c.session.Document.SetBeadColor(row, col, c.session.SelectedColor.ID)
	case model.ToolEraser:
		err = c.session.Document.ClearBead(row, col)
	case model.ToolMark:
		err = c.session.Document.ToggleCompleted(row, col)
	default:
		err = fmt.Errorf("unsupported tool %q", c.session.CurrentTool)
	}
	if err != nil {
		c.logger.Error("bead activation failed", "row", row, "column", col, "error", err)
		return err
	}
	c.session.Dirty = true
	c.logger.Info("bead updated", "row", row, "column", col, "tool", c.session.CurrentTool)
	c.notify()
	return nil
}

func (c *Controller) RemoveSelectedRow() error {
	if c.session.Selection.Mode != model.SelectionRow {
		return errors.New("no row selected")
	}
	if err := c.session.Document.RemoveRow(c.session.Selection.Index); err != nil {
		return err
	}
	c.session.Selection = model.Selection{Mode: model.SelectionNone, Index: -1}
	c.session.Dirty = true
	c.logger.Info("row removed")
	c.notify()
	return nil
}

func (c *Controller) RemoveSelectedColumn() error {
	if c.session.Selection.Mode != model.SelectionColumn {
		return errors.New("no column selected")
	}
	if err := c.session.Document.RemoveColumn(c.session.Selection.Index); err != nil {
		return err
	}
	c.session.Selection = model.Selection{Mode: model.SelectionNone, Index: -1}
	c.session.Dirty = true
	c.logger.Info("column removed")
	c.notify()
	return nil
}

func (c *Controller) ZoomIn() {
	if c.session.Zoom >= MaxZoom {
		return
	}
	c.session.Zoom += 0.25
	if c.session.Zoom > MaxZoom {
		c.session.Zoom = MaxZoom
	}
	c.logger.Info("zoom changed", "zoom", c.session.Zoom)
	c.notify()
}

func (c *Controller) ZoomOut() {
	if c.session.Zoom <= MinZoom {
		return
	}
	c.session.Zoom -= 0.25
	if c.session.Zoom < MinZoom {
		c.session.Zoom = MinZoom
	}
	c.logger.Info("zoom changed", "zoom", c.session.Zoom)
	c.notify()
}

func (c *Controller) CanRemoveRow() bool {
	return c.session.Selection.Mode == model.SelectionRow && c.session.Document.Canvas.Height > 1
}

func (c *Controller) CanRemoveColumn() bool {
	return c.session.Selection.Mode == model.SelectionColumn && c.session.Document.Canvas.Width > 1
}

func chooseZoom(value float32) float32 {
	if value < MinZoom || value > MaxZoom {
		return 1
	}
	return value
}

func withPEYExtension(path string) string {
	if filepath.Ext(path) == ".pey" {
		return path
	}
	return path + ".pey"
}
