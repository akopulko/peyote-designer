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
	builder.WriteString("<table cellspacing=\"2\" cellpadding=\"0\" style=\"border-collapse:separate;\">")
	for _, row := range document.Beads {
		builder.WriteString("<tr>")
		for _, bead := range row {
			fill := "#FFFFFF"
			if bead.ColorID != "" {
				if color, ok := document.PaletteColorByID(bead.ColorID); ok {
					fill = color.Hex
				}
			}
			label := ""
			if bead.Completed {
				label = "&#10005;"
			}
			fmt.Fprintf(&builder, "<td style=\"width:22px;height:44px;border:1px solid #444;background:%s;text-align:center;\">%s</td>", fill, label)
		}
		builder.WriteString("</tr>")
	}
	builder.WriteString("</table></body></html>")

	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
