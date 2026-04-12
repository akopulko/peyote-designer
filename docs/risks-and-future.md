# Risks and Future Enhancements

## Initial Risks

- Fyne desktop printing support is limited, so the MVP uses a first-pass export-oriented print flow.
- Large bracelet maps may require a more optimized renderer if documents grow significantly.
- Windows builds require native or MinGW-backed CGO support, so local cross-compilation depends on the extra toolchain being installed. CI builds Windows arm64 natively on GitHub's `windows-11-arm` runner with LLVM's `clang` toolchain.

## Future Enhancements

- import bracelet images and map colors to the nearest palette
- richer print preview and native print integration
- undo and redo support
- named palettes and palette import/export
- recent files and autosave
- keyboard shortcuts for common editing actions
