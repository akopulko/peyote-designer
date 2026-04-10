package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	application "github.com/kostya/peyote-designer/internal/app"
	"github.com/kostya/peyote-designer/internal/buildinfo"
	applog "github.com/kostya/peyote-designer/internal/logging"
	"github.com/kostya/peyote-designer/internal/persistence"
	"github.com/kostya/peyote-designer/internal/printing"
	"github.com/kostya/peyote-designer/internal/ui"
)

func main() {
	fyneApp := app.NewWithID("com.kostya.peyote-designer")
	if icon := loadAppIcon(); icon != nil {
		fyneApp.SetIcon(icon)
	}
	logBuffer := applog.NewBuffer(500)
	logger := applog.NewLogger(logBuffer)
	store := persistence.NewStore()
	controller, err := application.NewController(store, logger)
	if err != nil {
		panic(err)
	}
	printer := printing.NewFilePrinter()

	window := ui.NewMainWindow(fyneApp, controller, logger, logBuffer, printer)
	logger.Info("application started", "version", buildinfo.DisplayVersion(), "commit", buildinfo.Commit, "build_date", buildinfo.BuildDate)
	window.ShowAndRun()
}

func loadAppIcon() fyne.Resource {
	paths := []string{
		filepath.Join("icons", "app.png"),
		filepath.Join("Resources", "app.png"),
		filepath.Join("..", "Resources", "app.png"),
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths,
			filepath.Join(exeDir, "app.png"),
			filepath.Join(exeDir, "..", "Resources", "app.png"),
		)
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return &fyne.StaticResource{
				StaticName:    "app.png",
				StaticContent: data,
			}
		}
	}
	return nil
}
