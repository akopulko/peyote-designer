package main

import (
	"fyne.io/fyne/v2/app"

	application "github.com/kostya/peyote-designer/internal/app"
	applog "github.com/kostya/peyote-designer/internal/logging"
	"github.com/kostya/peyote-designer/internal/persistence"
	"github.com/kostya/peyote-designer/internal/printing"
	"github.com/kostya/peyote-designer/internal/ui"
)

func main() {
	fyneApp := app.NewWithID("com.kostya.peyote-designer")
	logBuffer := applog.NewBuffer(500)
	logger := applog.NewLogger(logBuffer)
	store := persistence.NewStore()
	controller, err := application.NewController(store, logger)
	if err != nil {
		panic(err)
	}
	printer := printing.NewFilePrinter()

	window := ui.NewMainWindow(fyneApp, controller, logger, logBuffer, printer)
	logger.Info("application started")
	window.ShowAndRun()
}
