package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/kostya/peyote-designer/internal/importing"
)

const (
	initialImportPreviewWidth  = 640
	initialImportPreviewHeight = 520
)

func (mw *MainWindow) showImportDialog() {
	if mw.importer == nil {
		dialog.ShowError(fmt.Errorf("image import is not available"), mw.window)
		return
	}
	mw.confirmImportIfNeeded(mw.showImportFilePicker)
}

func (mw *MainWindow) confirmImportIfNeeded(next func()) {
	if !mw.controller.Session().Dirty {
		next()
		return
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Unsaved Changes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Import will open the imported result as a new unsaved file."),
		widget.NewLabel("Save the current file before continuing, continue without saving, or cancel import."),
	)
	importDialog := dialog.NewCustomWithoutButtons("Import Image", content, mw.window)
	saveButton := widget.NewButtonWithIcon("Save and Continue", theme.DocumentSaveIcon(), func() {
		importDialog.Hide()
		mw.saveDocumentWithCompletion(next)
	})
	saveButton.Importance = widget.HighImportance
	continueButton := widget.NewButton("Continue Without Saving", func() {
		importDialog.Hide()
		next()
	})
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), importDialog.Hide)
	importDialog.SetButtons([]fyne.CanvasObject{
		cancelButton,
		continueButton,
		saveButton,
	})
	importDialog.Show()
}

func (mw *MainWindow) showImportFilePicker() {
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

		source, err := mw.importer.LoadImage(path)
		if err != nil {
			mw.logger.Error("image import load failed", "path", path, "error", err)
			dialog.ShowInformation("Import Image", importing.FriendlyError(err), mw.window)
			return
		}
		mw.showImportAreaSelection(source)
	}, mw.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
	fd.Resize(fyne.NewSize(1000, 720))
	fd.Show()
}

func (mw *MainWindow) showImportAreaSelection(source *importing.SourceImage) {
	selector := newCropSelectionWidget(source)
	nextButton := widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), nil)
	nextButton.Importance = widget.HighImportance
	selectionLabel := widget.NewLabel(selectionText(selector.Selection()))

	selectionWindow := mw.app.NewWindow("Select Import Area")
	selectionWindow.Resize(fyne.NewSize(1080, 780))

	selector.onChanged = func(selection image.Rectangle, valid bool) {
		selectionLabel.SetText(selectionText(selection))
		if valid {
			nextButton.Enable()
		} else {
			nextButton.Disable()
		}
	}
	nextButton.OnTapped = func() {
		selection := selector.Selection()
		if selection.Dx() <= 0 || selection.Dy() <= 0 {
			dialog.ShowInformation("Import Image", importing.FriendlyError(importing.ErrInvalidSelection), selectionWindow)
			return
		}
		selectionWindow.Close()
		mw.showImportConfiguration(source, selection)
	}
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), selectionWindow.Close)
	selectionWindow.SetContent(container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Drag on the image to choose the area to import."),
			selectionLabel,
		),
		container.NewHBox(layout.NewSpacer(), cancelButton, nextButton),
		nil,
		nil,
		container.NewScroll(selector),
	))
	selectionWindow.Show()
}

func (mw *MainWindow) showImportConfiguration(source *importing.SourceImage, selection image.Rectangle) {
	config := importing.Config{
		BeadCount:  importing.DefaultBeadCount,
		ColorCount: importing.DefaultColorCount,
	}
	gridWidth, gridHeight, _ := mw.importer.GridSize(selection, config.BeadCount)
	preview := newImportPreviewWidget(source, selection, gridWidth, gridHeight)
	beadEntry := widget.NewEntry()
	beadEntry.SetText(strconv.Itoa(config.BeadCount))
	colourEntry := widget.NewEntry()
	colourEntry.SetText(strconv.Itoa(config.ColorCount))
	gridLabel := widget.NewLabel("")
	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord
	importButton := widget.NewButtonWithIcon("Import", theme.UploadIcon(), nil)
	importButton.Importance = widget.HighImportance

	update := func() {
		beadCount, beadErr := strconv.Atoi(strings.TrimSpace(beadEntry.Text))
		colourCount, colourErr := strconv.Atoi(strings.TrimSpace(colourEntry.Text))
		var err error
		if beadErr != nil {
			err = fmt.Errorf("enter a bead count between %d and %d",
				importing.MinBeadCount,
				importing.MaxBeadCount,
			)
		}
		if err == nil && colourErr != nil {
			err = fmt.Errorf("enter a colour count between 0 and %d", importing.MaxColorCount)
		}
		if err == nil {
			gridWidth, gridHeight, err = mw.importer.GridSize(selection, beadCount)
		}
		if err == nil && (colourCount < 0 || colourCount > importing.MaxColorCount) {
			err = fmt.Errorf("enter a colour count between 0 and %d", importing.MaxColorCount)
		}
		if err != nil {
			errorLabel.SetText(importing.FriendlyError(err))
			importButton.Disable()
			return
		}

		config.BeadCount = beadCount
		config.ColorCount = colourCount
		preview.SetGrid(gridWidth, gridHeight)
		colourText := strconv.Itoa(config.ColorCount)
		if config.ColorCount == 0 {
			colourText = "Auto"
		}
		gridLabel.SetText(fmt.Sprintf("%d x %d = %d beads, %s colours",
			gridWidth,
			gridHeight,
			gridWidth*gridHeight,
			colourText,
		))
		errorLabel.SetText("")
		importButton.Enable()
	}

	beadEntry.OnChanged = func(string) {
		update()
	}
	colourEntry.OnChanged = func(string) {
		update()
	}
	beadMinus := widget.NewButton("-", func() {
		stepEntry(beadEntry, -1, importing.MinBeadCount, importing.MaxBeadCount)
	})
	beadPlus := widget.NewButton("+", func() {
		stepEntry(beadEntry, 1, importing.MinBeadCount, importing.MaxBeadCount)
	})
	colourMinus := widget.NewButton("-", func() {
		stepEntry(colourEntry, -1, 0, importing.MaxColorCount)
	})
	colourPlus := widget.NewButton("+", func() {
		stepEntry(colourEntry, 1, 0, importing.MaxColorCount)
	})

	previewScroll := container.NewScroll(preview)
	previewScroll.SetMinSize(fyne.NewSize(initialImportPreviewWidth, initialImportPreviewHeight))
	previewControls := container.NewHBox(
		widget.NewButtonWithIcon("Zoom Out", theme.ZoomOutIcon(), preview.ZoomOut),
		widget.NewButtonWithIcon("Zoom In", theme.ZoomInIcon(), preview.ZoomIn),
	)
	leftPane := container.NewBorder(
		nil,
		previewControls,
		nil,
		nil,
		previewScroll,
	)
	controls := container.NewVBox(
		widget.NewLabelWithStyle("Import Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Total beads"),
		container.NewBorder(nil, nil, beadMinus, beadPlus, beadEntry),
		widget.NewLabel("Colours"),
		container.NewBorder(nil, nil, colourMinus, colourPlus, colourEntry),
		gridLabel,
		errorLabel,
		layout.NewSpacer(),
	)
	content := container.NewBorder(nil, nil, nil, controls, leftPane)
	configWindow := mw.app.NewWindow("Configure Import")
	importButton.OnTapped = func() {
		doc, err := mw.importer.Convert(source, selection, config)
		if err != nil {
			mw.logger.Error("image import conversion failed", "path", source.Path, "error", err)
			dialog.ShowInformation("Import Image", importing.FriendlyError(err), configWindow)
			return
		}
		if err := mw.controller.LoadImportedDocument(doc, source.Path); err != nil {
			mw.logger.Error("image import document load failed", "path", source.Path, "error", err)
			dialog.ShowError(err, configWindow)
			return
		}
		configWindow.Close()
	}
	cancelButton := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), configWindow.Close)
	configWindow.SetContent(container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), cancelButton, importButton),
		nil,
		nil,
		content,
	))
	configWindow.Resize(fyne.NewSize(1120, 780))
	update()
	configWindow.Show()
}

func stepEntry(entry *widget.Entry, delta, min, max int) {
	value, err := strconv.Atoi(strings.TrimSpace(entry.Text))
	if err != nil {
		value = min
	}
	value = clampUIInt(value+delta, min, max)
	entry.SetText(strconv.Itoa(value))
}

func selectionText(selection image.Rectangle) string {
	if selection.Dx() <= 0 || selection.Dy() <= 0 {
		return "No area selected."
	}
	return fmt.Sprintf("Selected area: %d x %d pixels", selection.Dx(), selection.Dy())
}

type cropSelectionWidget struct {
	widget.BaseWidget
	source    *importing.SourceImage
	raster    *canvas.Raster
	selection image.Rectangle
	dragStart image.Point
	dragging  bool
	onChanged func(image.Rectangle, bool)
}

func newCropSelectionWidget(source *importing.SourceImage) *cropSelectionWidget {
	w := &cropSelectionWidget{
		source:    source,
		selection: source.Bounds,
	}
	w.raster = canvas.NewRaster(w.render)
	w.raster.SetMinSize(fyne.NewSize(980, 620))
	w.ExtendBaseWidget(w)
	return w
}

func (w *cropSelectionWidget) Selection() image.Rectangle {
	return w.selection.Canon().Intersect(w.source.Bounds)
}

func (w *cropSelectionWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.raster)
}

func (w *cropSelectionWidget) MinSize() fyne.Size {
	return w.raster.MinSize()
}

func (w *cropSelectionWidget) MouseDown(event *desktop.MouseEvent) {
	w.dragging = true
	w.dragStart = w.imagePoint(event.Position)
	w.selection = image.Rect(w.dragStart.X, w.dragStart.Y, w.dragStart.X+1, w.dragStart.Y+1).Intersect(w.source.Bounds)
	w.notifySelection()
	w.Refresh()
}

func (w *cropSelectionWidget) MouseUp(*desktop.MouseEvent) {
	w.dragging = false
}

func (w *cropSelectionWidget) Dragged(event *fyne.DragEvent) {
	if !w.dragging {
		return
	}
	point := w.imagePoint(event.Position)
	w.selection = image.Rect(w.dragStart.X, w.dragStart.Y, point.X, point.Y).Canon().Intersect(w.source.Bounds)
	w.notifySelection()
	w.Refresh()
}

func (w *cropSelectionWidget) DragEnd() {
	w.dragging = false
}

func (w *cropSelectionWidget) notifySelection() {
	if w.onChanged != nil {
		selection := w.Selection()
		w.onChanged(selection, selection.Dx() > 0 && selection.Dy() > 0)
	}
}

func (w *cropSelectionWidget) imagePoint(position fyne.Position) image.Point {
	display := fitRect(w.raster.Size(), w.source.Bounds.Dx(), w.source.Bounds.Dy())
	if display.Dx() <= 0 || display.Dy() <= 0 {
		return w.source.Bounds.Min
	}
	x := float64(position.X) - float64(display.Min.X)
	y := float64(position.Y) - float64(display.Min.Y)
	srcX := w.source.Bounds.Min.X + int(x*float64(w.source.Bounds.Dx())/float64(display.Dx()))
	srcY := w.source.Bounds.Min.Y + int(y*float64(w.source.Bounds.Dy())/float64(display.Dy()))
	return image.Pt(
		clampUIInt(srcX, w.source.Bounds.Min.X, w.source.Bounds.Max.X),
		clampUIInt(srcY, w.source.Bounds.Min.Y, w.source.Bounds.Max.Y),
	)
}

func (w *cropSelectionWidget) render(width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.NRGBA{R: 242, G: 242, B: 242, A: 255}}, image.Point{}, draw.Src)
	display := fitRect(fyne.NewSize(float32(width), float32(height)), w.source.Bounds.Dx(), w.source.Bounds.Dy())
	drawImage(dst, w.source.Image, w.source.Bounds, display)
	drawSelectionOverlay(dst, w.source.Bounds, display, w.Selection())
	return dst
}

type importPreviewWidget struct {
	widget.BaseWidget
	source    *importing.SourceImage
	selection image.Rectangle
	raster    *canvas.Raster
	gridWidth int
	gridHigh  int
	zoom      float32
}

func newImportPreviewWidget(
	source *importing.SourceImage,
	selection image.Rectangle,
	gridWidth int,
	gridHeight int,
) *importPreviewWidget {
	w := &importPreviewWidget{
		source:    source,
		selection: selection,
		gridWidth: gridWidth,
		gridHigh:  gridHeight,
		zoom:      1,
	}
	w.raster = canvas.NewRaster(w.render)
	w.updateMinSize()
	w.ExtendBaseWidget(w)
	return w
}

func (w *importPreviewWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.raster)
}

func (w *importPreviewWidget) MinSize() fyne.Size {
	return w.raster.MinSize()
}

func (w *importPreviewWidget) SetGrid(width, height int) {
	w.gridWidth = width
	w.gridHigh = height
	w.Refresh()
}

func (w *importPreviewWidget) ZoomIn() {
	w.zoom = minUIFloat(w.zoom+0.25, 4)
	w.updateMinSize()
	w.Refresh()
}

func (w *importPreviewWidget) ZoomOut() {
	w.zoom = maxUIFloat(w.zoom-0.25, 0.5)
	w.updateMinSize()
	w.Refresh()
}

func (w *importPreviewWidget) updateMinSize() {
	w.raster.SetMinSize(fyne.NewSize(
		initialImportPreviewWidth*w.zoom,
		initialImportPreviewHeight*w.zoom,
	))
}

func (w *importPreviewWidget) render(width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.NRGBA{R: 245, G: 245, B: 245, A: 255}}, image.Point{}, draw.Src)
	display := fitRect(fyne.NewSize(float32(width), float32(height)), w.selection.Dx(), w.selection.Dy())
	drawImage(dst, w.source.Image, w.selection, display)
	drawGrid(dst, display, w.gridWidth, w.gridHigh)
	return dst
}

func fitRect(size fyne.Size, sourceWidth, sourceHeight int) image.Rectangle {
	if sourceWidth <= 0 || sourceHeight <= 0 || size.Width <= 0 || size.Height <= 0 {
		return image.Rectangle{}
	}
	scale := math.Min(float64(size.Width)/float64(sourceWidth), float64(size.Height)/float64(sourceHeight))
	width := maxUIInt(1, int(math.Round(float64(sourceWidth)*scale)))
	height := maxUIInt(1, int(math.Round(float64(sourceHeight)*scale)))
	x := int(math.Round((float64(size.Width) - float64(width)) / 2))
	y := int(math.Round((float64(size.Height) - float64(height)) / 2))
	return image.Rect(x, y, x+width, y+height)
}

func drawImage(dst *image.RGBA, src image.Image, srcRect image.Rectangle, dstRect image.Rectangle) {
	if dstRect.Dx() <= 0 || dstRect.Dy() <= 0 || srcRect.Dx() <= 0 || srcRect.Dy() <= 0 {
		return
	}
	for y := dstRect.Min.Y; y < dstRect.Max.Y; y++ {
		srcY := srcRect.Min.Y + int(float64(y-dstRect.Min.Y)*float64(srcRect.Dy())/float64(dstRect.Dy()))
		srcY = clampUIInt(srcY, srcRect.Min.Y, srcRect.Max.Y-1)
		for x := dstRect.Min.X; x < dstRect.Max.X; x++ {
			srcX := srcRect.Min.X + int(float64(x-dstRect.Min.X)*float64(srcRect.Dx())/float64(dstRect.Dx()))
			srcX = clampUIInt(srcX, srcRect.Min.X, srcRect.Max.X-1)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
}

func drawSelectionOverlay(dst *image.RGBA, imageBounds image.Rectangle, display image.Rectangle, selection image.Rectangle) {
	if display.Dx() <= 0 || display.Dy() <= 0 || selection.Dx() <= 0 || selection.Dy() <= 0 {
		return
	}
	selected := image.Rect(
		display.Min.X+int(float64(selection.Min.X-imageBounds.Min.X)*float64(display.Dx())/float64(imageBounds.Dx())),
		display.Min.Y+int(float64(selection.Min.Y-imageBounds.Min.Y)*float64(display.Dy())/float64(imageBounds.Dy())),
		display.Min.X+int(float64(selection.Max.X-imageBounds.Min.X)*float64(display.Dx())/float64(imageBounds.Dx())),
		display.Min.Y+int(float64(selection.Max.Y-imageBounds.Min.Y)*float64(display.Dy())/float64(imageBounds.Dy())),
	)

	overlay := color.NRGBA{R: 0, G: 0, B: 0, A: 80}
	fillOverlay(dst, image.Rect(display.Min.X, display.Min.Y, display.Max.X, selected.Min.Y), overlay)
	fillOverlay(dst, image.Rect(display.Min.X, selected.Max.Y, display.Max.X, display.Max.Y), overlay)
	fillOverlay(dst, image.Rect(display.Min.X, selected.Min.Y, selected.Min.X, selected.Max.Y), overlay)
	fillOverlay(dst, image.Rect(selected.Max.X, selected.Min.Y, display.Max.X, selected.Max.Y), overlay)
	strokeRectUI(dst, selected, color.NRGBA{R: 20, G: 120, B: 220, A: 255}, 3)
}

func drawGrid(dst *image.RGBA, rect image.Rectangle, columns, rows int) {
	if columns <= 0 || rows <= 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	halo := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	stroke := color.NRGBA{R: 0, G: 72, B: 140, A: 255}
	for col := 0; col <= columns; col++ {
		x := rect.Min.X + int(math.Round(float64(col)*float64(rect.Dx())/float64(columns)))
		drawLineThicknessUI(dst, x, rect.Min.Y, x, rect.Max.Y-1, halo, 3)
		drawLineThicknessUI(dst, x, rect.Min.Y, x, rect.Max.Y-1, stroke, 1)
	}
	for row := 0; row <= rows; row++ {
		y := rect.Min.Y + int(math.Round(float64(row)*float64(rect.Dy())/float64(rows)))
		drawLineThicknessUI(dst, rect.Min.X, y, rect.Max.X-1, y, halo, 3)
		drawLineThicknessUI(dst, rect.Min.X, y, rect.Max.X-1, y, stroke, 1)
	}
	strokeRectUI(dst, rect, halo, 4)
	strokeRectUI(dst, rect, stroke, 2)
}

func fillOverlay(dst *image.RGBA, rect image.Rectangle, overlay color.NRGBA) {
	rect = rect.Intersect(dst.Bounds())
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			base := dst.RGBAAt(x, y)
			dst.Set(x, y, alphaBlend(base, overlay))
		}
	}
}

func alphaBlend(base color.RGBA, overlay color.NRGBA) color.NRGBA {
	alpha := int(overlay.A)
	invAlpha := 255 - alpha
	return color.NRGBA{
		R: uint8((int(base.R)*invAlpha + int(overlay.R)*alpha) / 255),
		G: uint8((int(base.G)*invAlpha + int(overlay.G)*alpha) / 255),
		B: uint8((int(base.B)*invAlpha + int(overlay.B)*alpha) / 255),
		A: 255,
	}
}

func strokeRectUI(dst *image.RGBA, rect image.Rectangle, stroke color.NRGBA, thickness int) {
	for inset := 0; inset < thickness; inset++ {
		current := image.Rect(rect.Min.X+inset, rect.Min.Y+inset, rect.Max.X-inset, rect.Max.Y-inset)
		if current.Dx() <= 0 || current.Dy() <= 0 {
			return
		}
		for x := current.Min.X; x < current.Max.X; x++ {
			setPixel(dst, x, current.Min.Y, stroke)
			setPixel(dst, x, current.Max.Y-1, stroke)
		}
		for y := current.Min.Y; y < current.Max.Y; y++ {
			setPixel(dst, current.Min.X, y, stroke)
			setPixel(dst, current.Max.X-1, y, stroke)
		}
	}
}

func drawLineUI(dst *image.RGBA, x1, y1, x2, y2 int, stroke color.NRGBA) {
	if x1 == x2 {
		from := minUIInt(y1, y2)
		to := maxUIInt(y1, y2)
		for y := from; y <= to; y++ {
			setPixel(dst, x1, y, stroke)
		}
		return
	}
	from := minUIInt(x1, x2)
	to := maxUIInt(x1, x2)
	for x := from; x <= to; x++ {
		setPixel(dst, x, y1, stroke)
	}
}

func drawLineThicknessUI(dst *image.RGBA, x1, y1, x2, y2 int, stroke color.NRGBA, thickness int) {
	radius := thickness / 2
	if x1 == x2 {
		for offset := -radius; offset <= radius; offset++ {
			drawLineUI(dst, x1+offset, y1, x2+offset, y2, stroke)
		}
		return
	}
	for offset := -radius; offset <= radius; offset++ {
		drawLineUI(dst, x1, y1+offset, x2, y2+offset, stroke)
	}
}

func setPixel(dst *image.RGBA, x, y int, c color.NRGBA) {
	if image.Pt(x, y).In(dst.Bounds()) {
		dst.Set(x, y, c)
	}
}

func clampUIInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minUIInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxUIInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minUIFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxUIFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
