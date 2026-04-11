# UI Layout Proposal

## Main Window

- top grouped icon-only toolbar with two rows for scaled Windows displays
- left scrollable bead map canvas
- right information panel
- native-style application menu

## Toolbar Groups

- document actions: new, open, save, import image, print
- selection actions: select row, select column, remove row, remove column
- edit actions: paint, set colour, eraser, mark
- view actions: zoom in, zoom out

The toolbar is split into a document/selection row and a bead/tool row so actions remain visible and clickable when the app is maximized on high-DPI Windows displays.

The active paint tool and active selection mode are visually emphasized.

## Right Panel

Shows:

- total bead count
- completed bead count
- incomplete bead count
- palette usage summary with swatch, index, hex, and usage count

## Interaction Model

- bead clicks act on the current selection mode first, then on the active tool
- only one document is open at a time
- dirty-document prompts gate new, open, close, and import flows
