# Upstream provenance

This directory is a source snapshot of
[`jellevandenhooff/gosim`](https://github.com/jellevandenhooff/gosim).

- Upstream commit: `ffd3a613542675755e4cbf8186b5edaf404ed95c`
- Upstream branch: `main`
- Imported: 2026-08-09

The upstream `.git` directory is intentionally excluded. The original
`go.mod` is retained so gosim remains a nested module and does not add its
dependencies to the Temporal server module.

Local compatibility changes should be kept small and documented here so a
future upstream refresh can distinguish them from the imported snapshot.
