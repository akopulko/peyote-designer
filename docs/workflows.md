# GitHub Actions and Release Workflow Proposal

## Pull Request Validation

Run on pull requests:

- `go mod tidy` and fail on diff
- `go test ./...`
- `golangci-lint run ./...`

## Branch Build Workflow

Run on pushes to `main` and `develop`:

- build the macOS app bundle on `macos-latest` and package `peyote-designer-macos-arm64.dmg`
- build the Windows binary on `windows-latest` and package `peyote-designer-windows-amd64.zip`
- upload artifacts for inspection

## Tagged Release Workflow

Run on tags matching `v*`:

- build and package `peyote-designer-macos-arm64.dmg` on `macos-latest`
- build and package `peyote-designer-windows-amd64.zip` on `windows-latest`
- attach both artifacts to the GitHub release
- keep `.goreleaser.yml` in the repository as a baseline release config for future expansion
