# Explicit Struct-Based Golden File Assertions

Status: accepted (generated, pending human review)

## Context

Traditional golden file packages either register global CLI flags into flag.CommandLine, rely on global singletons, or lack line ending and volatile token normalization. This leads to flag collision in large test suites and broken assertions across operating systems (such as Windows CRLF line endings).

## Decision

We encapsulate golden file options (Dir, Update, Normalizers) in an explicit `Golden` struct and accept a minimal `TB` interface (`Helper()`, `Errorf()`, `Fatalf()`). Names are strictly sanitized against path traversal, writes are atomic, and LCS line diffs are bounded to 4,000,000 matrix cells.

## Consequences

Tests control update behavior explicitly without flag conflicts or global state. Golden comparisons run deterministically across platforms with clear failure messages.
