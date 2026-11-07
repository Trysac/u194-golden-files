# Testing

Tests are located in `goldenfiles_test.go`. They verify functionality using both `*testing.T` and a recording fake implementation of the `goldenfiles.TB` interface.

Coverage includes:
- Verification of matching golden files.
- Line diff mismatch formatting, including correct display of escaped control characters.
- Handling of missing golden files, including conditional updates via `Update` flag.
- Rejection of invalid golden file names (path traversal, absolute paths).
- Application of `NormalizeLineEndings` and `ReplaceRegexp` normalizers.
- Assertions for JSON data, including marshal failure handling.
- LCS diff bounds logic and trailing newline detection.

Tests are run with `-race` and are verified to pass on Windows filesystem paths.
