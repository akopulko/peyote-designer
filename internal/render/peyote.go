package render

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
)

func PeyoteMapSize(columns, rows int, metrics Metrics) image.Point {
	if columns <= 0 || rows <= 0 {
		return image.Point{}
	}
	width := metrics.Gap + columns*(metrics.BeadWidth+metrics.Gap)
	if rows > 1 {
		width += rowOffset(1, metrics)
	}
	height := metrics.Gap + rows*(metrics.BeadHeight+metrics.Gap)
	return image.Pt(width, height)
}

func PeyoteBeadBounds(row, col int, metrics Metrics) image.Rectangle {
	x := metrics.Gap + rowOffset(row, metrics) + col*(metrics.BeadWidth+metrics.Gap)
	y := metrics.Gap + row*(metrics.BeadHeight+metrics.Gap)
	return image.Rect(x, y, x+metrics.BeadWidth, y+metrics.BeadHeight)
}

func HitTestPeyote(position fyne.Position, columns, rows int, metrics Metrics) (int, int, bool) {
	if columns <= 0 || rows <= 0 {
		return 0, 0, false
	}
	strideY := float32(metrics.BeadHeight + metrics.Gap)
	offsetY := position.Y - float32(metrics.Gap)
	if offsetY < 0 {
		return 0, 0, false
	}
	row := int(offsetY / strideY)
	if row < 0 || row >= rows {
		return 0, 0, false
	}

	strideX := float32(metrics.BeadWidth + metrics.Gap)
	offsetX := position.X - float32(metrics.Gap+rowOffset(row, metrics))
	if offsetX < 0 {
		return 0, 0, false
	}
	col := int(offsetX / strideX)
	if col < 0 || col >= columns {
		return 0, 0, false
	}

	rect := PeyoteBeadBounds(row, col, metrics)
	if !image.Pt(int(position.X), int(position.Y)).In(rect) {
		return 0, 0, false
	}
	return row, col, true
}

func DrawPeyoteBead(img *image.RGBA, rect image.Rectangle, fill color.NRGBA, stroke color.NRGBA) {
	fillRect(img, rect, fill)
	strokeRect(img, rect, stroke)
}

func DrawPeyoteBeadOutline(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA, thickness int) {
	for i := 0; i < thickness; i++ {
		inner := insetRect(rect, i)
		if inner.Dx() <= 0 || inner.Dy() <= 0 {
			return
		}
		strokeRect(img, inner, stroke)
	}
}

func DrawPeyoteOverlay(dst *image.RGBA, rect image.Rectangle, columns, rows int, stroke color.NRGBA, halo color.NRGBA) {
	if columns <= 0 || rows <= 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	logicalColumns := float64(columns)
	if rows > 1 {
		logicalColumns += 0.5
	}
	strideX := float64(rect.Dx()) / logicalColumns
	strideY := float64(rect.Dy()) / float64(rows)
	for row := 0; row < rows; row++ {
		rowOffset := 0.0
		if row%2 == 1 {
			rowOffset = strideX / 2
		}
		for col := 0; col < columns; col++ {
			beadRect := image.Rect(
				rect.Min.X+int(math.Round(rowOffset+float64(col)*strideX)),
				rect.Min.Y+int(math.Round(float64(row)*strideY)),
				rect.Min.X+int(math.Round(rowOffset+float64(col+1)*strideX)),
				rect.Min.Y+int(math.Round(float64(row+1)*strideY)),
			)
			DrawPeyoteBeadOutline(dst, beadRect, halo, 3)
			DrawPeyoteBeadOutline(dst, beadRect, stroke, 1)
		}
	}
}

func rowOffset(row int, metrics Metrics) int {
	if row%2 == 0 {
		return 0
	}
	return (metrics.BeadWidth + metrics.Gap) / 2
}

func fillRect(img *image.RGBA, rect image.Rectangle, fill color.NRGBA) {
	rect = rect.Intersect(img.Bounds())
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, fill)
		}
	}
}

func strokeRect(img *image.RGBA, rect image.Rectangle, stroke color.NRGBA) {
	rect = rect.Intersect(img.Bounds())
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, stroke)
		img.Set(x, rect.Max.Y-1, stroke)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, stroke)
		img.Set(rect.Max.X-1, y, stroke)
	}
}

func insetRect(rect image.Rectangle, inset int) image.Rectangle {
	return image.Rect(rect.Min.X+inset, rect.Min.Y+inset, rect.Max.X-inset, rect.Max.Y-inset)
}
