# `.pey` Data Model Proposal

## Schema

```json
{
  "version": 1,
  "metadata": {
    "appName": "Peyote Designer",
    "title": "Sample Bracelet",
    "createdAt": "2026-04-09T10:00:00Z",
    "updatedAt": "2026-04-09T10:15:00Z"
  },
  "canvas": {
    "width": 8,
    "height": 12
  },
  "palette": [
    {
      "id": "color-1",
      "index": 1,
      "name": "Red",
      "hex": "#D73A31"
    }
  ],
  "beads": [
    [
      {
        "colorId": "color-1",
        "completed": false
      }
    ]
  ],
  "view": {
    "zoom": 1,
    "selectedTool": "paint",
    "selectedColorId": "color-1"
  },
  "extensions": {}
}
```

## Rules

- `version` starts at `1`.
- `beads` is row-major and must match `canvas.height x canvas.width`.
- `palette[*].index` is stable and used for display summaries.
- beads reference palette entries by `colorId`.
- empty beads omit `colorId` and render white.
- `extensions` is reserved for future import and export metadata.

