# Peyote Designer

Peyote Designer is a desktop application for creating, editing, tracking, and printing peyote stitch bracelet patterns. It is built in Go with `fyne.io` and stores bracelet projects as JSON-based `.pey` files.

## Current MVP

The initial implementation includes:

- single-document editing
- new, open, save, and save as flows
- peyote bead map rendering with zoom
- paint, eraser, and mark tools
- row and column selection and removal
- right-hand project statistics and palette summary
- structured application logging with an in-app debug log window
- import placeholder and first-pass print/export flow

## Repository Structure

- `cmd/peyote-designer/`: application entry point and dependency wiring
- `internal/app/`: document session and controller logic
- `internal/model/`: core domain model and bead map operations
- `internal/persistence/`: `.pey` JSON loading and saving
- `internal/render/`: bead map widget and zoom metrics
- `internal/ui/`: Fyne window, menus, toolbar, dialogs, and panels
- `internal/logging/`: structured logger and in-memory debug buffer
- `internal/printing/`: print/export service
- `internal/importing/`: import extension point placeholder
- `docs/`: architecture, schema, UI, workflow, and roadmap documents
- `sample-data/`: example `.pey` files
- `.github/workflows/`: CI, branch build, and tagged release workflows

## Local Development

Requirements:

- Go 1.26+
- `golangci-lint` for `make lint`
- platform GUI prerequisites required by Fyne

Common commands:

```bash
make run
make test
make build-macos
make build-windows
make package
```

## Build and Packaging

- `make build-macos` creates `dist/peyote-designer-darwin-arm64`
- `make build-windows` creates `dist/peyote-designer-windows-amd64.exe`
- `make package` creates ZIP archives for both targets in `dist/`

`make build` currently aliases the macOS build for local desktop work.

Windows builds require either:

- a native Windows environment, or
- `x86_64-w64-mingw32-gcc` installed locally for cross-compilation

## Releases

GitHub Actions handles three workflows:

- pull request validation
- native branch builds for `main` and `develop`
- tagged releases for `v*`

Tagged releases build native macOS and Windows binaries on their respective runners, package ZIP archives, and attach them to the GitHub release.

## File Format Overview

Projects use the `.pey` extension and are stored as JSON. Each file contains:

- schema version and metadata
- bracelet width and height
- ordered palette definitions
- row-major bead data with color references and completion state
- optional view preferences
- extension space for future image import metadata

See [docs/data-model.md](docs/data-model.md) for the full schema proposal.
