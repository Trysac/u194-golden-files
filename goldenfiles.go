package goldenfiles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxGoldenFileSize is the maximum allowed size for a golden file (64 MiB).
const MaxGoldenFileSize = 64 * 1024 * 1024

// MaxDiffMatrixCells is the product threshold for line counts above which LCS diff is skipped.
const MaxDiffMatrixCells = 4_000_000

var validNameSegment = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// TB is the interface required for reporting test failures and registering test helpers.
// It is satisfied by *testing.T and *testing.B.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Normalizer transforms bytes before comparison or update.
type Normalizer func([]byte) []byte

// Golden manages golden file assertions and updates.
type Golden struct {
	// Dir specifies the directory containing golden files. Defaults to "testdata" when empty.
	Dir string
	// Update indicates whether golden files should be updated with normalized got bytes.
	Update bool
	// Normalizers are applied sequentially to both got bytes and stored golden bytes.
	Normalizers []Normalizer
}

// ValidateName checks that name is safe: 1 to 200 bytes, slash-separated segments
// matching ^[A-Za-z0-9._-]+$, with no ".." segments.
func ValidateName(name string) error {
	if len(name) < 1 || len(name) > 200 {
		return errors.New("length out of bounds")
	}
	if strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return errors.New("invalid path separators or root/trailing slash")
	}
	segments := strings.Split(name, "/")
	for _, seg := range segments {
		if seg == "" || seg == ".." || seg == "." || !validNameSegment.MatchString(seg) {
			return errors.New("invalid segment")
		}
	}
	return nil
}

func (g Golden) targetDir() string {
	if g.Dir == "" {
		return "testdata"
	}
	return g.Dir
}

// Assert asserts that got matches the golden file for name, applying normalizers in order.
// When Update is true, got is normalized and written to the golden file atomically.
func (g Golden) Assert(t TB, name string, got []byte) {
	t.Helper()
	if err := ValidateName(name); err != nil {
		t.Fatalf("invalid golden name %q", name)
		return
	}

	dir := g.targetDir()
	relPath := filepath.FromSlash(name) + ".golden"
	path := filepath.Join(dir, relPath)

	normGot := got
	for _, norm := range g.Normalizers {
		if norm != nil {
			normGot = norm(normGot)
		}
	}

	if g.Update {
		d := filepath.Dir(path)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("cannot create directory %s: %v", d, err)
			return
		}
		tmpFile, err := os.CreateTemp(d, ".golden-tmp-*")
		if err != nil {
			t.Fatalf("cannot create temp file in %s: %v", d, err)
			return
		}
		tmpName := tmpFile.Name()
		if _, err := tmpFile.Write(normGot); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
			t.Fatalf("cannot write temp file %s: %v", tmpName, err)
			return
		}
		if err := tmpFile.Chmod(0o644); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
			t.Fatalf("cannot chmod temp file %s: %v", tmpName, err)
			return
		}
		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpName)
			t.Fatalf("cannot close temp file %s: %v", tmpName, err)
			return
		}
		if err := os.Rename(tmpName, path); err != nil {
			_ = os.Remove(tmpName)
			t.Fatalf("cannot rename %s to %s: %v", tmpName, path, err)
			return
		}
		return
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s missing; rerun with GOLDEN_UPDATE=1", path)
			return
		}
		t.Fatalf("cannot open golden file %s: %v", path, err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err == nil && fi.Size() > MaxGoldenFileSize {
		t.Fatalf("golden file %s too large", path)
		return
	}

	limitReader := io.LimitReader(f, MaxGoldenFileSize+1)
	stored, err := io.ReadAll(limitReader)
	if err != nil {
		t.Fatalf("cannot read golden file %s: %v", path, err)
		return
	}
	if int64(len(stored)) > MaxGoldenFileSize {
		t.Fatalf("golden file %s too large", path)
		return
	}

	normWant := stored
	for _, norm := range g.Normalizers {
		if norm != nil {
			normWant = norm(normWant)
		}
	}

	diff := Diff(normWant, normGot)
	if diff != "" {
		t.Errorf("golden mismatch for %s (%s):\n%s", name, path, diff)
	}
}

// AssertJSON marshals v using json.MarshalIndent with two spaces plus a trailing newline,
// then delegates to Assert.
func (g Golden) AssertJSON(t TB, name string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("cannot marshal value for %s", name)
		return
	}
	data = append(data, '\n')
	g.Assert(t, name, data)
}

// NormalizeLineEndings replaces CRLF ("\r\n") and lone CR ("\r") with LF ("\n").
func NormalizeLineEndings(b []byte) []byte {
	if len(b) == 0 {
		return []byte{}
	}
	res := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\r' {
			if i+1 < len(b) && b[i+1] == '\n' {
				i++
			}
			res = append(res, '\n')
		} else {
			res = append(res, b[i])
		}
	}
	return res
}

// ReplaceRegexp returns a Normalizer that substitutes occurrences of re with repl.
func ReplaceRegexp(re *regexp.Regexp, repl string) Normalizer {
	return func(b []byte) []byte {
		if re == nil {
			return b
		}
		return re.ReplaceAll(b, []byte(repl))
	}
}

// escapeLine escapes '\r' and other control bytes except tab ('\t') using \r or \xNN notation.
func escapeLine(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\t':
			sb.WriteByte('\t')
		case c == '\r':
			sb.WriteString(`\r`)
		case c < 0x20 || c == 0x7f:
			fmt.Fprintf(&sb, `\x%02x`, c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

func splitLines(b []byte) ([]string, bool) {
	if len(b) == 0 {
		return []string{}, false
	}
	hasTrailing := b[len(b)-1] == '\n'
	raw := b
	if hasTrailing {
		raw = b[:len(b)-1]
	}
	if len(raw) == 0 {
		return []string{""}, hasTrailing
	}
	parts := bytes.Split(raw, []byte("\n"))
	lines := make([]string, len(parts))
	for i, p := range parts {
		lines[i] = string(p)
	}
	return lines, hasTrailing
}

type diffOp int

const (
	opEqual diffOp = iota
	opDelete
	opInsert
)

type diffEntry struct {
	op   diffOp
	line string
	num  int // 1-based original line number in want (delete) or got (insert)
}

// Diff computes a line-by-line diff between want and got.
// Equal inputs return "".
// When want and got differ only in their trailing newline, "files differ only in trailing newline\n" is returned.
// When lines count product exceeds 4,000,000, "files differ (diff skipped: too large)\n" is returned.
func Diff(want, got []byte) string {
	if bytes.Equal(want, got) {
		return ""
	}

	wantLines, wantTrailing := splitLines(want)
	gotLines, gotTrailing := splitLines(got)

	// Check if content matches but trailing newline differs
	if wantTrailing != gotTrailing && len(wantLines) == len(gotLines) {
		matches := true
		for i := range wantLines {
			if wantLines[i] != gotLines[i] {
				matches = false
				break
			}
		}
		if matches {
			return "files differ only in trailing newline\n"
		}
	}

	n := len(wantLines)
	m := len(gotLines)
	if int64(n)*int64(m) > MaxDiffMatrixCells {
		return "files differ (diff skipped: too large)\n"
	}

	// LCS table
	dist := make([][]int, n+1)
	for i := 0; i <= n; i++ {
		dist[i] = make([]int, m+1)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if wantLines[i] == gotLines[j] {
				dist[i+1][j+1] = dist[i][j] + 1
			} else if dist[i+1][j] >= dist[i][j+1] {
				dist[i+1][j+1] = dist[i+1][j]
			} else {
				dist[i+1][j+1] = dist[i][j+1]
			}
		}
	}

	// Backtrack diff entries
	var entries []diffEntry
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && wantLines[i-1] == gotLines[j-1] {
			entries = append(entries, diffEntry{op: opEqual, line: wantLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dist[i][j-1] >= dist[i-1][j]) {
			entries = append(entries, diffEntry{op: opInsert, line: gotLines[j-1], num: j})
			j--
		} else if i > 0 && (j == 0 || dist[i][j-1] < dist[i-1][j]) {
			entries = append(entries, diffEntry{op: opDelete, line: wantLines[i-1], num: i})
			i--
		}
	}

	// Reverse entries to chronological order
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}

	// Group into contiguous change hunks and output deletions before insertions per hunk
	var sb strings.Builder
	k := 0
	for k < len(entries) {
		if entries[k].op == opEqual {
			k++
			continue
		}
		hunkStart := k
		for k < len(entries) && entries[k].op != opEqual {
			k++
		}
		hunk := entries[hunkStart:k]
		// Output deletions first
		for _, item := range hunk {
			if item.op == opDelete {
				sb.WriteString(fmt.Sprintf("-%d: %s\n", item.num, escapeLine(item.line)))
			}
		}
		// Then insertions
		for _, item := range hunk {
			if item.op == opInsert {
				sb.WriteString(fmt.Sprintf("+%d: %s\n", item.num, escapeLine(item.line)))
			}
		}
	}

	if wantTrailing != gotTrailing {
		if sb.Len() > 0 {
			sb.WriteString("files differ in trailing newline\n")
		} else {
			sb.WriteString("files differ only in trailing newline\n")
		}
	}

	return sb.String()
}
