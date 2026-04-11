package printing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kostya/peyote-designer/internal/model"
)

type Printer interface {
	Print(document *model.Document) (string, error)
}

type FilePrinter struct{}

func NewFilePrinter() *FilePrinter {
	return &FilePrinter{}
}

func (p *FilePrinter) Print(document *model.Document) (string, error) {
	name := strings.ReplaceAll(strings.TrimSpace(document.Metadata.Title), " ", "-")
	if name == "" {
		name = "peyote-pattern"
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.html", name, time.Now().UnixNano()))

	stats := document.Stats()
	var builder strings.Builder
	builder.WriteString("<html><body style=\"font-family:sans-serif;max-width:800px;margin:24px;\">")
	builder.WriteString("<h1>Peyote Designer Print Preview</h1>")
	fmt.Fprintf(&builder, "<p>Size: %d x %d beads</p>", document.Canvas.Width, document.Canvas.Height)
	fmt.Fprintf(&builder, "<p>Total: %d, Completed: %d, Incomplete: %d</p>", stats.Total, stats.Completed, stats.Incomplete)
	builder.WriteString("<div style=\"display:inline-block;padding:8px;background:#f5f3ee;\">")
	for rowIndex, row := range document.Beads {
		marginLeft := 0
		if rowIndex%2 == 1 {
			marginLeft = 13
		}
		fmt.Fprintf(&builder, "<div style=\"display:flex;gap:4px;margin-left:%dpx;margin-bottom:4px;\">", marginLeft)
		for _, bead := range row {
			fill := "#FFFFFF"
			if bead.ColorID != "" {
				if color, ok := document.PaletteColorByID(bead.ColorID); ok {
					fill = model.NormalizeHex(color.Hex)
				}
			}
			label := ""
			if bead.Completed {
				label = "&#10005;"
			}
			fmt.Fprintf(
				&builder,
				"<div style=\"width:22px;height:33px;border:1px solid #444;background:%s;text-align:center;line-height:33px;\">%s</div>",
				fill,
				label,
			)
		}
		builder.WriteString("</div>")
	}
	builder.WriteString("</div></body></html>")

	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
