# golden-files

[![Go Reference](https://pkg.go.dev/badge/github.com/Trysac/u194-golden-files.svg)](https://pkg.go.dev/github.com/Trysac/u194-golden-files)

Deterministic, dependency-free golden file assertions for Go test suites. Go tests that compare large outputs to expected files often reimplement helpers that lack portability or readability. This package provides a single, testable assertion helper that supports explicit update control, line-ending normalization, and readable diffs.

## Installation

```bash
go get github.com/Trysac/u194-golden-files
```

## Usage

```go
package mypackage_test

import (
	"os"
	"testing"
	"github.com/Trysac/u194-golden-files"
)

func TestOutput(t *testing.T) {
	g := goldenfiles.Golden{Update: os.Getenv("GOLDEN_UPDATE") == "1"}
	g.Assert(t, "my_output", []byte("hello world\n"))
}
```

## Public API

- `type TB interface { Helper(); Errorf(string, ...any); Fatalf(string, ...any) }`: Interface satisfied by `*testing.T`, allowing the use of recording fakes.
- `type Normalizer func([]byte) []byte`: Function type for modifying input before assertion or storage.
- `type Golden struct { Dir string; Update bool; Normalizers []Normalizer }`: Configuration for assertions. `Dir` defaults to "testdata".
- `Assert(t TB, name string, got []byte)`: Compares `got` against a golden file. Normalizers are applied to both input and file data. Files are capped at 64 MiB.
- `AssertJSON(t TB, name string, v any)`: Marshals `v` with two-space indentation and asserts against a golden file.
- `Diff(want, got []byte) string`: Generates a human-readable line-by-line diff. It detects trailing newline mismatches and escapes control characters. Diffs are skipped if the product of line counts exceeds 4,000,000.
- `NormalizeLineEndings(b []byte) []byte`: Converts CRLF and CR to LF.
- `ReplaceRegexp(re *regexp.Regexp, repl string) Normalizer`: Returns a `Normalizer` that replaces regex matches in the input.

## Why not goldie or gotest.tools/golden?

Existing alternatives often rely on global flags (e.g., `flag.CommandLine`), which creates collisions in complex test suites. `golden-files` uses an explicit struct configuration, avoiding global state and side effects, and uses an injected `TB` interface to support easier testing of the helper itself.

## Security

See [docs/SECURITY.md](docs/SECURITY.md) for threat modeling and constraints.
