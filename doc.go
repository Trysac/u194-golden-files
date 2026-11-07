// Package goldenfiles is a small, dependency-free Go library: Golden file assertions for Go tests: update via a flag field, normalizers for line endings and volatile text, line diffs and safe names
//
// The public API is every exported identifier of this root package. Functions are pure: they do
// no I/O, keep no global mutable state and report invalid input through returned errors.
package goldenfiles
