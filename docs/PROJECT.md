# Project

## Objective
Provide a lightweight, dependency-free, and deterministic golden file assertion library for Go. This package eliminates repetitive test code by standardizing golden file reads, atomic updates, and readable diff generation across Go repositories.

## Scope
- Supporting `*testing.T` and `*testing.B` via an injected interface.
- Normalizing test outputs (line endings, volatile regex-based tokens).
- Atomic file writes using temporary files and `os.Rename`.
- Generating readable, line-based diffs for mismatches.
- JSON golden file assertions.

## Out of Scope
- Global flags management.
- Managing snapshot directories per individual test name.
- JSON-aware structural diffing.
- Output coloring.

## Factory run u194 (measured from API usage)

| Phase | Model | Batch | Input | Output | Cache write | Cache read | Cost (USD) |
|---|---|---|---|---|---|---|---|
| generation | gemini-3.8-flash | yes | 4017 | 303 | 0 | 0 | 0.0021 |
| generation | gemini-3.8-flash | yes | 4017 | 12384 | 0 | 0 | 0.0247 |
| fix-1 | gemini-3.1-flash-lite | no | 24536 | 411 | 0 | 24383 | 0.0074 |
| audit | gemini-3.1-flash-lite | no | 23059 | 87 | 0 | 18328 | 0.0064 |
| docs | gemini-3.1-flash-lite | no | 3104 | 1801 | 0 | 0 | 0.0035 |

Total: $0.0440. Prices: Anthropic list prices.
