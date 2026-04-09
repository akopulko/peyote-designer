# GitHub Actions and Release Workflow Proposal

## Pull Request Validation

Run on pull requests:

- `go mod tidy` and fail on diff
- `go test ./...`
- `golangci-lint run ./...`

## Branch Build Workflow

Run on pushes to `main` and `develop`:

- build macOS binary on `macos-latest`
- build Windows binary on `windows-latest`
- upload artifacts for inspection

## Tagged Release Workflow

Run on tags matching `v*`:

- build and zip the macOS binary on `macos-latest`
- build and zip the Windows binary on `windows-latest`
- attach both ZIP archives to the GitHub release
- keep `.goreleaser.yml` in the repository as a baseline release config for future expansion
