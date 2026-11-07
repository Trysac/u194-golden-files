# Architecture

`goldenfiles` provides golden file comparison and update logic for Go tests.

## Design Goals

1. **Zero External Dependencies**: Implemented strictly with the Go standard library.
2. **Explicit Update Control**: Avoids registering global flags to prevent collisions. Update behavior is configured via `goldenfiles.Golden{Update: bool}`.
3. **Safe File Operations**: Names are validated against a strict regex; `..` is disallowed. Files are capped at 64 MiB. Updates are atomic, written through directory-local temporary files.
4. **Portable Comparisons**: Normalizers handle line-ending differences (CRLF/CR to LF). Diff output uses Myers/LCS algorithm to produce readable line-based differences.
