# Security

## Threat Model

### Path Traversal
Golden file names are validated to ensure they are safe for file system operations. They must match the regex `^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$` and must be between 1 and 200 bytes. Components such as `..`, absolute paths (e.g., `/abs`), and Windows reserved characters (e.g., `a\b`, `x:y`) are explicitly rejected with `Fatalf` to prevent directory traversal or invalid file operations.

### Resource Exhaustion
- **Input Size**: Golden file reads are capped at 64 MiB. Any file exceeding this size triggers a `Fatalf` to prevent memory exhaustion.
- **Diff Complexity**: If the product of the line count of the expectation and the reality exceeds 4,000,000, the diff algorithm is bypassed, and the library returns a "diff skipped" message to prevent excessive CPU and memory consumption during comparisons.
