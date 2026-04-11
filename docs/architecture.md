# Architecture

Peyote Designer uses a layered MVVM-style desktop architecture with a single in-memory document session.

## Layers

- `cmd/peyote-designer`: starts the Fyne application and wires services.
- `internal/app`: owns the active session, command handling, dirty state, zoom state, selection state, and document lifecycle.
- `internal/model`: defines the document schema, bead map operations, palette management, and derived statistics.
- `internal/persistence`: persists `.pey` files to and from JSON.
- `internal/ui`: owns Fyne widgets, menus, dialogs, toolbar composition, and panel refresh logic.
- `internal/render`: renders the bead map through a dedicated widget with hit-testing and zoom-aware metrics.
- `internal/logging`: exposes structured logging and an in-memory ring buffer for the debug log window.
- `internal/printing`: encapsulates the first-pass print/export workflow.
- `internal/importing`: decodes supported image files and converts selected raster regions into bead-map documents.

## State Management

The controller keeps a single active `Session` that contains:

- active `Document`
- current file path
- dirty flag
- active paint tool
- pending selection mode
- current selection
- selected color
- zoom level

UI components subscribe to controller changes and redraw from the session snapshot.

## Extension Points

- `printing.Printer` isolates print/export details from the UI.
- `importing.Service` isolates image decoding, grid sizing, colour reduction, and document generation from the UI flow.
- `Document.Extensions` reserves schema space for future import metadata without breaking current files.
