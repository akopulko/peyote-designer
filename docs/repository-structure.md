# Repository Structure Proposal

```text
.
├── .github/workflows/
├── cmd/peyote-designer/
├── docs/
├── icons/
├── internal/
│   ├── app/
│   ├── importing/
│   ├── logging/
│   ├── model/
│   ├── persistence/
│   ├── printing/
│   ├── render/
│   └── ui/
├── sample-data/
├── assets/
├── Makefile
└── README.md
```

Notes:

- `internal/` holds all non-public application code.
- `pkg/` is intentionally omitted because the application does not expose a reusable library yet.
- `assets/` and `icons/` exist for future packaged resources even though the MVP mostly relies on Fyne theme icons.

