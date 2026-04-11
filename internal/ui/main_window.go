package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"math"
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
	"github.com/kostya/peyote-designer/internal/buildinfo"
	"github.com/kostya/peyote-designer/internal/importing"
	applog "github.com/kostya/peyote-designer/internal/logging"
	"github.com/kostya/peyote-designer/internal/model"
	"github.com/kostya/peyote-designer/internal/printing"
	"github.com/kostya/peyote-designer/internal/render"
)

type MainWindow struct {
	app           fyne.App
	window        fyne.Window
	controller    *app.Controller
	logger        *slog.Logger
	logBuffer     *applog.Buffer
	printer       printing.Printer
	importer      *importing.Service
	beadMap       *render.BeadMap
	scroll        *container.Scroll
	hScroll       *widget.Slider
	vScroll       *widget.Slider
	statsLabel    *widget.Label
	paletteBox    *fyne.Container
	colorPreview  *canvas.Rectangle
	toolButtons   map[model.Tool]*widget.Button
	selectRowBtn  *widget.Button
	selectColBtn  *widget.Button
	removeBtn     *widget.Button
	resizeBtn     *widget.Button
	debugWindow   fyne.Window
	syncingScroll bool
	mainMenu      *fyne.MainMenu
	resizeItem    *fyne.MenuItem
	removeRowItem *fyne.MenuItem
	removeColItem *fyne.MenuItem
}

func NewMainWindow(
	fyneApp fyne.App,
	controller *app.Controller,
	logger *slog.Logger,
	logBuffer *applog.Buffer,
	printer printing.Printer,
	importer *importing.Service,
) *MainWindow {
	window := fyneApp.NewWindow(model.AppName)
	window.Resize(fyne.NewSize(1280, 840))

	mw := &MainWindow{
		app:          fyneApp,
		window:       window,
		controller:   controller,
		logger:       logger,
		logBuffer:    logBuffer,
		printer:      printer,
		importer:     importer,
		statsLabel:   widget.NewLabel(""),
		paletteBox:   container.NewVBox(),
		colorPreview: canvas.NewRectangle(theme.Color(theme.ColorNamePrimary)),
		toolButtons:  make(map[model.Tool]*widget.Button),
	}

	mw.colorPreview.SetMinSize(fyne.NewSize(18, 18))
	mw.beadMap = render.NewBeadMap(controller)
	mw.scroll = container.NewScroll(mw.beadMap)
	mw.scroll.Direction = container.ScrollBoth
	mw.scroll.SetMinSize(fyne.NewSize(860, 700))
	mw.hScroll = widget.NewSlider(0, 1)
	mw.hScroll.Step = 1
	mw.hScroll.OnChanged = func(value float64) {
		mw.applyScrollFromSliders(false, value)
	}
	mw.vScroll = widget.NewSlider(0, 1)
	mw.vScroll.Orientation = widget.Vertical
	mw.vScroll.Step = 1
	mw.vScroll.OnChanged = func(value float64) {
		mw.applyScrollFromSliders(true, value)
	}
	mw.scroll.OnScrolled = func(position fyne.Position) {
		mw.syncScrollControls(position)
	}

	rightPanel := mw.buildRightPanel()
	toolbar := mw.buildToolbar()
	scrollArea := container.NewBorder(nil, mw.hScroll, nil, mw.vScroll, mw.scroll)
	content := container.NewBorder(toolbar, nil, nil, rightPanel, scrollArea)

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
	mw.resizeItem = fyne.NewMenuItem("Resize", mw.showResizeDialog)
	mw.removeRowItem = fyne.NewMenuItem("Remove Beads Row", mw.removeRow)
	mw.removeColItem = fyne.NewMenuItem("Remove Beads Column", mw.removeColumn)
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New", mw.showNewDialog),
		fyne.NewMenuItem("Open", mw.showOpenDialog),
		fyne.NewMenuItem("Save", mw.saveDocument),
		fyne.NewMenuItem("Save As", mw.showSaveDialog),
		fyne.NewMenuItem("Import Image", mw.showImportDialog),
		fyne.NewMenuItem("Print", mw.printDocument),
	)
	editMenu := fyne.NewMenu("Edit",
		mw.resizeItem,
		fyne.NewMenuItem("Select Row", func() { mw.controller.SetSelectionTarget(model.SelectionRow) }),
		fyne.NewMenuItem("Select Column", func() { mw.controller.SetSelectionTarget(model.SelectionColumn) }),
		mw.removeRowItem,
		mw.removeColItem,
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
	mw.mainMenu = fyne.NewMainMenu(fileMenu, editMenu, toolsMenu, helpMenu)
	return mw.mainMenu
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
	importButton := makeButton(theme.FileImageIcon(), mw.showImportDialog)
	printButton := makeButton(theme.DocumentPrintIcon(), mw.printDocument)
	mw.resizeBtn = makeButton(theme.ViewFullScreenIcon(), mw.showResizeDialog)
	mw.selectRowBtn = makeButton(theme.MoreHorizontalIcon(), func() { mw.controller.SetSelectionTarget(model.SelectionRow) })
	mw.selectColBtn = makeButton(theme.MoreVerticalIcon(), func() { mw.controller.SetSelectionTarget(model.SelectionColumn) })
	mw.removeBtn = makeButton(theme.CancelIcon(), mw.removeSelection)
	zoomInButton := makeButton(theme.ZoomInIcon(), mw.controller.ZoomIn)
	zoomOutButton := makeButton(theme.ZoomOutIcon(), mw.controller.ZoomOut)

	return container.NewHBox(
		newButton,
		openButton,
		saveButton,
		importButton,
		printButton,
		mw.resizeBtn,
		widget.NewSeparator(),
		mw.selectRowBtn,
		mw.selectColBtn,
		mw.removeBtn,
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
	paletteScroll := container.NewVScroll(mw.paletteBox)
	paletteScroll.SetMinSize(fyne.NewSize(300, 420))
	panel := container.NewVBox(
		widget.NewLabelWithStyle("Project Summary", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mw.statsLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Palette Summary", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		paletteScroll,
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
	mw.updateScrollbars()
	mw.updateButtonStates()
}

func (mw *MainWindow) updateButtonStates() {
	session := mw.controller.Session()
	hasDocument := mw.controller.HasDocument()
	for tool, button := range mw.toolButtons {
		if tool == session.CurrentTool && session.SelectionTarget == model.SelectionNone {
			button.Importance = widget.HighImportance
		} else {
			button.Importance = widget.MediumImportance
		}
		if hasDocument {
			button.Enable()
		} else {
			button.Disable()
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
	if hasDocument {
		mw.selectRowBtn.Enable()
		mw.selectColBtn.Enable()
		mw.resizeBtn.Enable()
	} else {
		mw.selectRowBtn.Disable()
		mw.selectColBtn.Disable()
		mw.resizeBtn.Disable()
	}
	mw.removeBtn.Disable()
	if mw.controller.CanRemoveRow() || mw.controller.CanRemoveColumn() {
		mw.removeBtn.Enable()
	}
	mw.updateMenuStates()
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

func (mw *MainWindow) showResizeDialog() {
	if !mw.controller.HasDocument() {
		dialog.ShowInformation("Resize", "Create or open a bracelet first.", mw.window)
		return
	}

	doc := mw.controller.Session().Document
	width := widget.NewEntry()
	width.SetText(strconv.Itoa(doc.Canvas.Width))
	height := widget.NewEntry()
	height.SetText(strconv.Itoa(doc.Canvas.Height))
	items := []*widget.FormItem{
		widget.NewFormItem("Width", width),
		widget.NewFormItem("Length", height),
	}
	dialog.ShowForm("Resize Bracelet", "Apply", "Cancel", items, func(ok bool) {
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
		if err := mw.controller.ResizeDocument(beadWidth, beadHeight); err != nil {
			dialog.ShowError(err, mw.window)
		}
	}, mw.window)
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
		fd.Resize(fyne.NewSize(1000, 720))
		fd.Show()
	})
}

func (mw *MainWindow) saveDocument() {
	mw.saveDocumentWithCompletion(nil)
}

func (mw *MainWindow) saveDocumentWithCompletion(afterSave func()) {
	if !mw.controller.HasDocument() {
		dialog.ShowInformation("Save", "Create or open a bracelet first.", mw.window)
		return
	}
	if mw.controller.Session().FilePath == "" {
		mw.showSaveDialogWithCompletion(afterSave)
		return
	}
	if err := mw.controller.Save(); err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	if afterSave != nil {
		afterSave()
	}
}

func (mw *MainWindow) showSaveDialog() {
	mw.showSaveDialogWithCompletion(nil)
}

func (mw *MainWindow) showSaveDialogWithCompletion(afterSave func()) {
	if !mw.controller.HasDocument() {
		dialog.ShowInformation("Save", "Create or open a bracelet first.", mw.window)
		return
	}
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
			return
		}
		if afterSave != nil {
			afterSave()
		}
	}, mw.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".pey"}))
	fd.SetFileName("bracelet.pey")
	fd.Resize(fyne.NewSize(1000, 720))
	fd.Show()
}

func (mw *MainWindow) showColorDialog() {
	presets := []string{
		"#000000", "#FFFFFF", "#D73A31", "#E7A600", "#2DA44E", "#1F6FEB",
		"#8250DF", "#BF3989", "#F66A0A", "#0969DA", "#0A3069", "#7C3AED",
		"#DB2777", "#DC2626", "#EA580C", "#CA8A04", "#16A34A", "#0891B2",
		"#2563EB", "#4F46E5", "#9333EA", "#C026D3", "#BE123C", "#78716C",
	}

	content := container.NewVBox(
		widget.NewLabel("Choose a preset colour or open the custom picker."),
		buildColorSwatchGrid(presets, mw.controller.Session().SelectedColor.Hex, func(hex string) {
			mw.controller.SetSelectedColor(hex)
		}),
	)

	customDialog := dialog.NewCustom("Select Colour", "Close", content, mw.window)
	customButton := widget.NewButton("Custom Colour…", func() {
		dialog.NewColorPicker("Custom Colour", "Choose the active bead colour", func(c color.Color) {
			r, g, b, _ := c.RGBA()
			mw.controller.SetSelectedColor(fmt.Sprintf("#%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8)))
			customDialog.Hide()
		}, mw.window).Show()
	})
	content.Add(customButton)
	customDialog.Show()
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

func (mw *MainWindow) removeSelection() {
	switch mw.controller.Session().Selection.Mode {
	case model.SelectionRow:
		mw.removeRow()
	case model.SelectionColumn:
		mw.removeColumn()
	default:
		dialog.ShowInformation("Remove", "Select a row or column first.", mw.window)
	}
}

func (mw *MainWindow) printDocument() {
	if !mw.controller.HasDocument() {
		dialog.ShowInformation("Print", "Create or open a bracelet first.", mw.window)
		return
	}
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
	logWindow := mw.app.NewWindow(fmt.Sprintf("Debug Log - %s %s", model.AppName, buildinfo.DisplayVersion()))
	versionLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Version %s", buildinfo.DisplayVersion()),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	entry := widget.NewRichText()
	entry.Wrapping = fyne.TextWrapWord
	refresh := func() {
		entries := mw.logBuffer.Entries()
		segments := make([]widget.RichTextSegment, 0, len(entries)*2)
		for _, item := range mw.logBuffer.Entries() {
			segments = append(segments, &widget.TextSegment{
				Text: item.Line,
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNameForeground,
					Inline:    false,
				},
			})
			segments = append(segments, &widget.TextSegment{
				Text: "\n",
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNameForeground,
					Inline:    false,
				},
			})
		}
		if len(entries) == 0 {
			segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text: "No log entries yet.",
					Style: widget.RichTextStyle{
						ColorName: theme.ColorNamePlaceHolder,
						Inline:    false,
					},
				},
			}
		}
		entry.Segments = segments
		entry.Refresh()
	}
	refresh()
	mw.logBuffer.Subscribe(func() {
		refresh()
	})
	clearButton := widget.NewButton("Clear", func() {
		mw.logBuffer.Clear()
	})
	copyButton := widget.NewButton("Copy", func() {
		lines := make([]string, 0, len(mw.logBuffer.Entries()))
		for _, item := range mw.logBuffer.Entries() {
			lines = append(lines, item.Line)
		}
		mw.app.Clipboard().SetContent(strings.Join(lines, "\n"))
	})
	logWindow.SetContent(container.NewBorder(versionLabel, container.NewHBox(clearButton, copyButton), nil, nil, container.NewScroll(entry)))
	logWindow.Resize(fyne.NewSize(760, 420))
	logWindow.SetOnClosed(func() {
		mw.debugWindow = nil
	})
	mw.debugWindow = logWindow
	logWindow.Show()
}

func (mw *MainWindow) updateMenuStates() {
	if mw.mainMenu == nil || mw.removeRowItem == nil || mw.removeColItem == nil || mw.resizeItem == nil {
		return
	}
	mw.resizeItem.Disabled = !mw.controller.HasDocument()
	mw.removeRowItem.Disabled = !mw.controller.CanRemoveRow()
	mw.removeColItem.Disabled = !mw.controller.CanRemoveColumn()
	mw.mainMenu.Refresh()
}

func (mw *MainWindow) updateScrollbars() {
	if !mw.controller.HasDocument() {
		mw.syncingScroll = true
		mw.hScroll.SetValue(0)
		mw.vScroll.SetValue(0)
		mw.syncingScroll = false
		mw.hScroll.Disable()
		mw.vScroll.Disable()
		return
	}

	contentSize := mw.beadMap.MinSize()
	viewSize := mw.scroll.Size()
	maxX := math.Max(0, float64(contentSize.Width-viewSize.Width))
	maxY := math.Max(0, float64(contentSize.Height-viewSize.Height))

	mw.syncingScroll = true
	mw.hScroll.Max = maxX
	mw.vScroll.Max = maxY
	mw.hScroll.Step = math.Max(1, maxX/50)
	mw.vScroll.Step = math.Max(1, maxY/50)
	mw.hScroll.SetValue(clampFloat(float64(mw.scroll.Offset.X), 0, maxX))
	mw.vScroll.SetValue(clampFloat(float64(mw.scroll.Offset.Y), 0, maxY))
	mw.syncingScroll = false

	if maxX > 0 {
		mw.hScroll.Enable()
	} else {
		mw.hScroll.Disable()
	}
	if maxY > 0 {
		mw.vScroll.Enable()
	} else {
		mw.vScroll.Disable()
	}
	mw.hScroll.Refresh()
	mw.vScroll.Refresh()
}

func (mw *MainWindow) syncScrollControls(position fyne.Position) {
	if mw.syncingScroll {
		return
	}
	mw.syncingScroll = true
	mw.hScroll.SetValue(float64(position.X))
	mw.vScroll.SetValue(float64(position.Y))
	mw.syncingScroll = false
}

func (mw *MainWindow) applyScrollFromSliders(vertical bool, value float64) {
	if mw.syncingScroll {
		return
	}
	offset := mw.scroll.Offset
	if vertical {
		offset.Y = float32(value)
	} else {
		offset.X = float32(value)
	}
	mw.scroll.ScrollToOffset(offset)
}

func buildStatsText(document *model.Document) string {
	if document == nil {
		return "No bracelet open.\nUse File > New or File > Open to begin."
	}
	stats := document.Stats()
	return fmt.Sprintf("Total beads: %d\nCompleted beads: %d\nIncomplete beads: %d", stats.Total, stats.Completed, stats.Incomplete)
}

func buildPaletteSummary(document *model.Document) []fyne.CanvasObject {
	if document == nil {
		return []fyne.CanvasObject{widget.NewLabel("No palette data yet.")}
	}
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
		return theme.Color(theme.ColorNamePrimary)
	}
	value, err := strconv.ParseUint(parsed, 16, 32)
	if err != nil {
		return theme.Color(theme.ColorNamePrimary)
	}
	return &color.NRGBA{
		R: uint8(value >> 16),
		G: uint8((value >> 8) & 0xFF),
		B: uint8(value & 0xFF),
		A: 0xFF,
	}
}

func windowTitle(session *app.Session) string {
	name := "No File Open"
	if session.FilePath != "" {
		name = filepath.Base(session.FilePath)
	} else if session.Document != nil && session.Document.Metadata.Title != "" {
		name = session.Document.Metadata.Title
	}
	if session.Dirty {
		name += " *"
	}
	return fmt.Sprintf("%s - %s", name, model.AppName)
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

type colorSwatch struct {
	widget.BaseWidget
	hex      string
	fill     color.Color
	selected bool
	tapped   func()
}

func newColorSwatch(hex string, selected bool, tapped func()) *colorSwatch {
	sw := &colorSwatch{
		hex:      hex,
		fill:     renderPreviewColor(hex),
		selected: selected,
		tapped:   tapped,
	}
	sw.ExtendBaseWidget(sw)
	return sw
}

func (s *colorSwatch) Tapped(*fyne.PointEvent) {
	if s.tapped != nil {
		s.tapped()
	}
}

func (s *colorSwatch) CreateRenderer() fyne.WidgetRenderer {
	fill := canvas.NewRectangle(s.fill)
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 2
	border.StrokeColor = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
	if s.selected {
		border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		border.StrokeWidth = 3
	}
	return &colorSwatchRenderer{swatch: s, fill: fill, border: border}
}

type colorSwatchRenderer struct {
	swatch *colorSwatch
	fill   *canvas.Rectangle
	border *canvas.Rectangle
}

func (r *colorSwatchRenderer) Layout(size fyne.Size) {
	r.fill.Resize(size)
	r.border.Resize(size)
}

func (r *colorSwatchRenderer) MinSize() fyne.Size {
	return fyne.NewSize(28, 28)
}

func (r *colorSwatchRenderer) Refresh() {
	r.fill.FillColor = r.swatch.fill
	r.fill.Refresh()
	r.border.StrokeColor = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
	r.border.StrokeWidth = 2
	if r.swatch.selected {
		r.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		r.border.StrokeWidth = 3
	}
	r.border.Refresh()
}

func (r *colorSwatchRenderer) Destroy() {}

func (r *colorSwatchRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.fill, r.border}
}

func buildColorSwatchGrid(colors []string, selectedHex string, onTap func(string)) fyne.CanvasObject {
	items := make([]fyne.CanvasObject, 0, len(colors))
	for _, hex := range colors {
		selected := strings.EqualFold(model.NormalizeHex(hex), model.NormalizeHex(selectedHex))
		current := hex
		items = append(items, newColorSwatch(current, selected, func() {
			onTap(current)
		}))
	}
	return container.NewGridWrap(fyne.NewSize(32, 32), items...)
}
