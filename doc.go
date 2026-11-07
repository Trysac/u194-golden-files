// Package goldenfiles provides deterministic, dependency-free golden file
// testing assertions for Go tests.
//
// Features include atomic snapshot updates via an explicit struct field (without
// requiring global command-line flags), built-in normalizers for line endings
// and volatile text (such as dates or IDs), path traversal protection, and
// readable LCS-based line diffs.
package goldenfiles
