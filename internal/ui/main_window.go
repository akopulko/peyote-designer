package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/kostya/peyote-designer/internal/app"
	applog "github.com/kostya/peyote-designer/internal/logging"
	"github.com/kostya/peyote-designer/internal/model"
	"github.com/kostya/peyote-designer/internal/printing"
	"github.com/kostya/peyote-designer/internal/render"
)

type MainWindow struct {
	app          fyne.App
	window       fyne.Window
	controller   *app.Controller
	logger       *slog.Logger
	logBuffer    *applog.Buffer
	printer      printing.Printer
	beadMap      *render.BeadMap
	scroll       *container.Scroll
	statsLabel   *widget.Label
	paletteBox   *fyne.Container
	colorPreview *canvas.Rectangle
	toolButtons  map[model.Tool]*widget.Button
	selectRowBtn *widget.Button
	selectColBtn *widget.Button
	removeRowBtn *widget.Button
	removeColBtn *widget.Button
	debugWindow  fyne.Window
}

func NewMainWindow(fyneApp fyne.App, controller *app.Controller, logger *slog.Logger, logBuffer *applog.Buffer, printer printing.Printer) *MainWindow {
	window := fyneApp.NewWindow(model.AppName)
	window.Resize(fyne.NewSize(1280, 840))

	mw := &MainWindow{
		app:          fyneApp,
		window:       window,
		controller:   controller,
		logger:       logger,
		logBuffer:    logBuffer,
		printer:      printer,
		statsLabel:   widget.NewLabel(""),
		paletteBox:   container.NewVBox(),
		colorPreview: canvas.NewRectangle(theme.PrimaryColor()),
		toolButtons:  make(map[model.Tool]*widget.Button),
	}

	mw.colorPreview.SetMinSize(fyne.NewSize(18, 18))
	mw.beadMap = render.NewBeadMap(controller)
	mw.scroll = container.NewScroll(mw.beadMap)
	mw.scroll.SetMinSize(fyne.NewSize(860, 700))

	rightPanel := mw.buildRightPanel()
	toolbar := mw.buildToolbar()
	content := container.NewBorder(toolbar, nil, nil, rightPanel, mw.scroll)

	window.SetContent(content)
	window.SetMainMenu(mw.buildMenu())
	window.SetCloseIntercept(func() {
		mw.confirmDiscardIfNeeded(func() {
			window.Close()
		})
	})

	controller.Subscribe(func() {
		mw.refresh()
	})
	mw.refresh()
	return mw
}

func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()
}

func (mw *MainWindow) buildMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New", mw.showNewDialog),
		fyne.NewMenuItem("Open", mw.showOpenDialog),
		fyne.NewMenuItem("Save", mw.saveDocument),
		fyne.NewMenuItem("Save As", mw.showSaveDialog),
		fyne.NewMenuItem("Import", mw.showImportPlaceholder),
		fyne.NewMenuItem("Print", mw.printDocument),
	)
	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Select Row", func() { mw.controller.SetSelectionTarget(model.SelectionRow) }),
		fyne.NewMenuItem("Select Column", func() { mw.controller.SetSelectionTarget(model.SelectionColumn) }),
		fyne.NewMenuItem("Remove Beads Row", mw.removeRow),
		fyne.NewMenuItem("Remove Beads Column", mw.removeColumn),
	)
	toolsMenu := fyne.NewMenu("Tools",
		fyne.NewMenuItem("Paint", func() { mw.controller.SetTool(model.ToolPaint) }),
		fyne.NewMenuItem("Set Colour", mw.showColorDialog),
		fyne.NewMenuItem("Eraser", func() { mw.controller.SetTool(model.ToolEraser) }),
		fyne.NewMenuItem("Mark", func() { mw.controller.SetTool(model.ToolMark) }),
		fyne.NewMenuItem("Zoom In", mw.controller.ZoomIn),
		fyne.NewMenuItem("Zoom Out", mw.controller.ZoomOut),
	)
	helpMenu := fyne.NewMenu("Help", fyne.NewMenuItem("Debug Log", mw.showDebugLog))
	return fyne.NewMainMenu(fileMenu, editMenu, toolsMenu, helpMenu)
}

func (mw *MainWindow) buildToolbar() fyne.CanvasObject {
	makeButton := func(icon fyne.Resource, tapped func()) *widget.Button {
		return widget.NewButtonWithIcon("", icon, tapped)
	}

	mw.toolButtons[model.ToolPaint] = makeButton(theme.ColorPaletteIcon(), func() { mw.controller.SetTool(model.ToolPaint) })
	mw.toolButtons[model.ToolEraser] = makeButton(theme.DeleteIcon(), func() { mw.controller.SetTool(model.ToolEraser) })
	mw.toolButtons[model.ToolMark] = makeButton(theme.ConfirmIcon(), func() { mw.controller.SetTool(model.ToolMark) })
	colorButton := makeButton(theme.ColorChromaticIcon(), mw.showColorDialog)
	newButton := makeButton(theme.DocumentCreateIcon(), mw.showNewDialog)
	openButton := makeButton(theme.FolderOpenIcon(), mw.showOpenDialog)
	saveButton := makeButton(theme.DocumentSaveIcon(), mw.saveDocument)
	printButton := makeButton(theme.DocumentPrintIcon(), mw.printDocument)
	mw.selectRowBtn = makeButton(theme.ContentAddIcon(), func() { mw.controller.SetSelectionTarget(model.SelectionRow) })
	mw.selectColBtn = makeButton(theme.MoreHorizontalIcon(), func() { mw.controller.SetSelectionTarget(model.SelectionColumn) })
	mw.removeRowBtn = makeButton(theme.ContentRemoveIcon(), mw.removeRow)
	mw.removeColBtn = makeButton(theme.ContentClearIcon(), mw.removeColumn)
	zoomInButton := makeButton(theme.ZoomInIcon(), mw.controller.ZoomIn)
	zoomOutButton := makeButton(theme.ZoomOutIcon(), mw.controller.ZoomOut)

	return container.NewHBox(
		newButton,
		openButton,
		saveButton,
		printButton,
		widget.NewSeparator(),
		mw.selectRowBtn,
		mw.selectColBtn,
		mw.removeRowBtn,
		mw.removeColBtn,
		widget.NewSeparator(),
		mw.toolButtons[model.ToolPaint],
		container.NewHBox(colorButton, mw.colorPreview),
		mw.toolButtons[model.ToolEraser],
		mw.toolButtons[model.ToolMark],
		widget.NewSeparator(),
		zoomInButton,
		zoomOutButton,
		layout.NewSpacer(),
	)
}

func (mw *MainWindow) buildRightPanel() fyne.CanvasObject {
	panel := container.NewVBox(
		widget.NewLabelWithStyle("Project Summary", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.statsLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Palette Summary", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.paletteBox,
	)
	return container.NewPadded(container.NewVBox(panel))
}

func (mw *MainWindow) refresh() {
	session := mw.controller.Session()
	mw.window.SetTitle(windowTitle(session))
	mw.statsLabel.SetText(buildStatsText(session.Document))
	mw.colorPreview.FillColor = renderPreviewColor(session.SelectedColor.Hex)
	mw.colorPreview.Refresh()
	mw.paletteBox.Objects = buildPaletteSummary(session.Document)
	mw.paletteBox.Refresh()
	mw.beadMap.Refresh()
	mw.scroll.Refresh()
	mw.updateButtonStates()
}

func (mw *MainWindow) updateButtonStates() {
	session := mw.controller.Session()
	for tool, button := range mw.toolButtons {
		if tool == session.CurrentTool && session.SelectionTarget == model.SelectionNone {
			button.Importance = widget.HighImportance
		} else {
			button.Importance = widget.MediumImportance
		}
		button.Refresh()
	}
	mw.selectRowBtn.Importance = widget.MediumImportance
	mw.selectColBtn.Importance = widget.MediumImportance
	if session.SelectionTarget == model.SelectionRow {
		mw.selectRowBtn.Importance = widget.HighImportance
	}
	if session.SelectionTarget == model.SelectionColumn {
		mw.selectColBtn.Importance = widget.HighImportance
	}
	mw.selectRowBtn.Refresh()
	mw.selectColBtn.Refresh()
	mw.removeRowBtn.Disable()
	mw.removeColBtn.Disable()
	if mw.controller.CanRemoveRow() {
		mw.removeRowBtn.Enable()
	}
	if mw.controller.CanRemoveColumn() {
		mw.removeColBtn.Enable()
	}
}

func (mw *MainWindow) showNewDialog() {
	mw.confirmDiscardIfNeeded(func() {
		width := widget.NewEntry()
		width.SetText("12")
		height := widget.NewEntry()
		height.SetText("20")
		items := []*widget.FormItem{
			widget.NewFormItem("Width", width),
			widget.NewFormItem("Height", height),
		}
		dialog.ShowForm("New Bracelet", "Create", "Cancel", items, func(ok bool) {
			if !ok {
				return
			}
			var beadWidth, beadHeight int
			if _, err := fmt.Sscanf(width.Text, "%d", &beadWidth); err != nil {
				dialog.ShowError(err, mw.window)
				return
			}
			if _, err := fmt.Sscanf(height.Text, "%d", &beadHeight); err != nil {
				dialog.ShowError(err, mw.window)
				return
			}
			if err := mw.controller.NewDocument(beadWidth, beadHeight); err != nil {
				dialog.ShowError(err, mw.window)
			}
		}, mw.window)
	})
}

func (mw *MainWindow) showOpenDialog() {
	mw.confirmDiscardIfNeeded(func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, mw.window)
				return
			}
			if reader == nil {
				return
			}
			path := reader.URI().Path()
			_ = reader.Close()
			if err := mw.controller.LoadDocument(path); err != nil {
				dialog.ShowError(err, mw.window)
			}
		}, mw.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".pey"}))
		fd.Show()
	})
}

func (mw *MainWindow) saveDocument() {
	if mw.controller.Session().FilePath == "" {
		mw.showSaveDialog()
		return
	}
	if err := mw.controller.Save(); err != nil {
		dialog.ShowError(err, mw.window)
	}
}

func (mw *MainWindow) showSaveDialog() {
	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, mw.window)
			return
		}
		if writer == nil {
			return
		}
		path := writer.URI().Path()
		_ = writer.Close()
		if err := mw.controller.SaveAs(path); err != nil {
			dialog.ShowError(err, mw.window)
		}
	}, mw.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".pey"}))
	fd.SetFileName("bracelet.pey")
	fd.Show()
}

func (mw *MainWindow) showColorDialog() {
	dialog.NewColorPicker("Set Colour", "Choose the active bead colour", func(c color.Color) {
		r, g, b, _ := c.RGBA()
		mw.controller.SetSelectedColor(fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8)))
	}, mw.window).Show()
}

func (mw *MainWindow) removeRow() {
	if err := mw.controller.RemoveSelectedRow(); err != nil {
		dialog.ShowError(err, mw.window)
	}
}

func (mw *MainWindow) removeColumn() {
	if err := mw.controller.RemoveSelectedColumn(); err != nil {
		dialog.ShowError(err, mw.window)
	}
}

func (mw *MainWindow) printDocument() {
	path, err := mw.printer.Print(mw.controller.Session().Document)
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	dialog.ShowInformation("Print Preview Generated", fmt.Sprintf("Generated preview at:\n%s", path), mw.window)
	if parsed, parseErr := url.Parse("file://" + path); parseErr == nil {
		_ = mw.app.OpenURL(parsed)
	}
}

func (mw *MainWindow) showImportPlaceholder() {
	dialog.ShowInformation("Import", "Image import is not implemented yet.", mw.window)
}

func (mw *MainWindow) confirmDiscardIfNeeded(next func()) {
	if !mw.controller.Session().Dirty {
		next()
		return
	}
	dialog.ShowConfirm("Unsaved Changes", "Discard current unsaved changes?", func(ok bool) {
		if ok {
			next()
		}
	}, mw.window)
}

func (mw *MainWindow) showDebugLog() {
	if mw.debugWindow != nil {
		mw.debugWindow.Show()
		mw.debugWindow.RequestFocus()
		return
	}
	logWindow := mw.app.NewWindow("Debug Log")
	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapWord
	entry.Disable()
	refresh := func() {
		lines := make([]string, 0, len(mw.logBuffer.Entries()))
		for _, item := range mw.logBuffer.Entries() {
			lines = append(lines, item.Line)
		}
		entry.SetText(strings.Join(lines, "\n"))
	}
	refresh()
	mw.logBuffer.Subscribe(func() {
		refresh()
	})
	clearButton := widget.NewButton("Clear", func() {
		mw.logBuffer.Clear()
	})
	copyButton := widget.NewButton("Copy", func() {
		mw.window.Clipboard().SetContent(entry.Text)
	})
	logWindow.SetContent(container.NewBorder(nil, container.NewHBox(clearButton, copyButton), nil, nil, entry))
	logWindow.Resize(fyne.NewSize(760, 420))
	logWindow.SetOnClosed(func() {
		mw.debugWindow = nil
	})
	mw.debugWindow = logWindow
	logWindow.Show()
}

func buildStatsText(document *model.Document) string {
	stats := document.Stats()
	return fmt.Sprintf("Total beads: %d\nCompleted beads: %d\nIncomplete beads: %d", stats.Total, stats.Completed, stats.Incomplete)
}

func buildPaletteSummary(document *model.Document) []fyne.CanvasObject {
	usage := document.PaletteUsage()
	if len(usage) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No colours used yet.")}
	}

	objects := make([]fyne.CanvasObject, 0, len(usage))
	for _, item := range usage {
		swatch := canvas.NewRectangle(renderPreviewColor(item.Color.Hex))
		swatch.SetMinSize(fyne.NewSize(18, 18))
		label := widget.NewLabel(fmt.Sprintf("#%d  %s  (%d beads)", item.Color.Index, item.Color.Hex, item.Count))
		objects = append(objects, container.NewHBox(swatch, label))
	}
	return objects
}

func renderPreviewColor(hex string) color.Color {
	return renderColor(hex)
}

func renderColor(hex string) color.Color {
	parsed := strings.TrimPrefix(model.NormalizeHex(hex), "#")
	if len(parsed) != 6 {
		return theme.PrimaryColor()
	}
	value, err := strconv.ParseUint(parsed, 16, 32)
	if err != nil {
		return theme.PrimaryColor()
	}
	return &color.NRGBA{
		R: uint8(value >> 16),
		G: uint8((value >> 8) & 0xFF),
		B: uint8(value & 0xFF),
		A: 0xFF,
	}
}

func windowTitle(session *app.Session) string {
	name := "Untitled"
	if session.FilePath != "" {
		name = filepath.Base(session.FilePath)
	} else if session.Document.Metadata.Title != "" {
		name = session.Document.Metadata.Title
	}
	if session.Dirty {
		name += " *"
	}
	return fmt.Sprintf("%s - %s", name, model.AppName)
}
