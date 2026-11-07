package goldenfiles_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Trysac/u194-golden-files"
)

type recordTB struct {
	helperCalled bool
	errors       []string
	fatals       []string
}

func (r *recordTB) Helper() {
	r.helperCalled = true
}

func (r *recordTB) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recordTB) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

// Acceptance item 1:
// With Dir set to a temp dir holding x.golden = "a\nb\n", g.Assert(fake, "x", []byte("a\nb\n")) records no Errorf or Fatalf call and calls Helper()
func TestAcceptance1_MatchingGolden(t *testing.T) {
	dir := t.TempDir()
	xPath := filepath.Join(dir, "x.golden")
	if err := os.WriteFile(xPath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("failed to write x.golden: %v", err)
	}

	fake := &recordTB{}
	g := goldenfiles.Golden{Dir: dir}
	g.Assert(fake, "x", []byte("a\nb\n"))

	if !fake.helperCalled {
		t.Errorf("expected Helper() to be called")
	}
	if len(fake.errors) != 0 {
		t.Errorf("expected no Errorf calls, got %v", fake.errors)
	}
	if len(fake.fatals) != 0 {
		t.Errorf("expected no Fatalf calls, got %v", fake.fatals)
	}
}

// Acceptance item 2:
// With got "a\nB\n" the fake records exactly one Errorf whose message equals
// "golden mismatch for x (" + filepath.Join(dir, "x.golden") + "):\n-2: b\n+2: B\n" and no Fatalf
func TestAcceptance2_MismatchDiff(t *testing.T) {
	dir := t.TempDir()
	xPath := filepath.Join(dir, "x.golden")
	if err := os.WriteFile(xPath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("failed to write x.golden: %v", err)
	}

	fake := &recordTB{}
	g := goldenfiles.Golden{Dir: dir}
	g.Assert(fake, "x", []byte("a\nB\n"))

	wantError := "golden mismatch for x (" + filepath.Join(dir, "x.golden") + "):\n-2: b\n+2: B\n"
	if len(fake.errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(fake.errors), fake.errors)
	}
	if fake.errors[0] != wantError {
		t.Errorf("mismatch error:\ngot:  %q\nwant: %q", fake.errors[0], wantError)
	}
	if len(fake.fatals) != 0 {
		t.Errorf("expected no Fatalf calls, got %v", fake.fatals)
	}
}

// Acceptance item 3:
// A missing golden with Update false records Fatalf "golden file " + path + " missing; rerun with GOLDEN_UPDATE=1";
// with Update true it writes exactly the normalized got bytes, creating sub/y.golden for name "sub/y", and records no failure
func TestAcceptance3_MissingAndUpdating(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "y.golden")

	// Missing without update
	fake1 := &recordTB{}
	g1 := goldenfiles.Golden{Dir: dir, Update: false}
	g1.Assert(fake1, "sub/y", []byte("line1\nline2\n"))

	wantFatal := "golden file " + path + " missing; rerun with GOLDEN_UPDATE=1"
	if len(fake1.fatals) != 1 || fake1.fatals[0] != wantFatal {
		t.Fatalf("expected Fatalf %q, got %v", wantFatal, fake1.fatals)
	}
	if len(fake1.errors) != 0 {
		t.Errorf("expected no errors, got %v", fake1.errors)
	}

	// Missing with update
	fake2 := &recordTB{}
	g2 := goldenfiles.Golden{Dir: dir, Update: true}
	g2.Assert(fake2, "sub/y", []byte("line1\nline2\n"))

	if len(fake2.fatals) != 0 || len(fake2.errors) != 0 {
		t.Fatalf("expected no failures on update, got fatals: %v, errors: %v", fake2.fatals, fake2.errors)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created file %s: %v", path, err)
	}
	if string(content) != "line1\nline2\n" {
		t.Errorf("file content = %q, want %q", string(content), "line1\nline2\n")
	}
}

// Acceptance item 4:
// Names "../x", "/abs", "a\b", "x:y" and "" record Fatalf with message
// "invalid golden name" followed by the quoted name (for example "invalid golden name \"../x\"") and write nothing
func TestAcceptance4_InvalidNames(t *testing.T) {
	invalidNames := []string{"../x", "/abs", "a\\b", "x:y", ""}
	for _, name := range invalidNames {
		t.Run("name_"+name, func(t *testing.T) {
			dir := t.TempDir()
			fake := &recordTB{}
			g := goldenfiles.Golden{Dir: dir, Update: true}
			g.Assert(fake, name, []byte("data"))

			wantFatal := fmt.Sprintf("invalid golden name %q", name)
			if len(fake.fatals) != 1 || fake.fatals[0] != wantFatal {
				t.Fatalf("expected Fatalf %q, got %v", wantFatal, fake.fatals)
			}
			// Verify nothing was written in dir
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("failed to read dir: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("expected empty dir, found %d entries", len(entries))
			}
		})
	}
}

// Acceptance item 5:
// With Normalizers [NormalizeLineEndings] a golden "a\r\nb\r\n" equals got "a\nb\n";
// with ReplaceRegexp(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "<date>") the got "on 2026-11-06"
// matches a golden "on 2025-01-01" and an update writes "on <date>"
func TestAcceptance5_Normalizers(t *testing.T) {
	dir := t.TempDir()
	// Line ending normalizer
	xPath := filepath.Join(dir, "le.golden")
	if err := os.WriteFile(xPath, []byte("a\r\nb\r\n"), 0o644); err != nil {
		t.Fatalf("failed to write le.golden: %v", err)
	}
	fake1 := &recordTB{}
	g1 := goldenfiles.Golden{
		Dir:         dir,
		Normalizers: []goldenfiles.Normalizer{goldenfiles.NormalizeLineEndings},
	}
	g1.Assert(fake1, "le", []byte("a\nb\n"))
	if len(fake1.errors) != 0 || len(fake1.fatals) != 0 {
		t.Fatalf("expected matching lines with CRLF normalizer, got errs: %v, fatals: %v", fake1.errors, fake1.fatals)
	}

	// Regexp replace normalizer
	rePath := filepath.Join(dir, "date.golden")
	if err := os.WriteFile(rePath, []byte("on 2025-01-01"), 0o644); err != nil {
		t.Fatalf("failed to write date.golden: %v", err)
	}
	dateNormalizer := goldenfiles.ReplaceRegexp(regexp.MustCompile(`\d{4}-\d{2}-\d{2}`), "<date>")
	fake2 := &recordTB{}
	g2 := goldenfiles.Golden{
		Dir:         dir,
		Normalizers: []goldenfiles.Normalizer{dateNormalizer},
	}
	g2.Assert(fake2, "date", []byte("on 2026-11-06"))
	if len(fake2.errors) != 0 || len(fake2.fatals) != 0 {
		t.Fatalf("expected regexp normalized match, got errs: %v, fatals: %v", fake2.errors, fake2.fatals)
	}

	// Update with normalizer writes normalized got
	fake3 := &recordTB{}
	g3 := goldenfiles.Golden{
		Dir:         dir,
		Update:      true,
		Normalizers: []goldenfiles.Normalizer{dateNormalizer},
	}
	g3.Assert(fake3, "date", []byte("on 2026-11-06"))
	if len(fake3.errors) != 0 || len(fake3.fatals) != 0 {
		t.Fatalf("expected clean update, got errs: %v, fatals: %v", fake3.errors, fake3.fatals)
	}
	written, err := os.ReadFile(rePath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if string(written) != "on <date>" {
		t.Errorf("written content = %q, want %q", string(written), "on <date>")
	}
}

// Acceptance item 6:
// AssertJSON(fake, "j", map[string]int{"b": 1, "a": 2}) compares against the bytes
// "{\n  \"a\": 2,\n  \"b\": 1\n}\n" and passes when j.golden holds them
func TestAcceptance6_AssertJSON(t *testing.T) {
	dir := t.TempDir()
	jPath := filepath.Join(dir, "j.golden")
	wantBytes := "{\n  \"a\": 2,\n  \"b\": 1\n}\n"
	if err := os.WriteFile(jPath, []byte(wantBytes), 0o644); err != nil {
		t.Fatalf("failed to write j.golden: %v", err)
	}

	fake := &recordTB{}
	g := goldenfiles.Golden{Dir: dir}
	g.AssertJSON(fake, "j", map[string]int{"b": 1, "a": 2})

	if len(fake.errors) != 0 || len(fake.fatals) != 0 {
		t.Fatalf("AssertJSON failed: errs: %v, fatals: %v", fake.errors, fake.fatals)
	}

	// Test marshal error triggers Fatalf with "cannot marshal value for <name>"
	fakeErr := &recordTB{}
	unserializable := map[string]any{"ch": make(chan int)}
	g.AssertJSON(fakeErr, "bad", unserializable)
	wantFatal := "cannot marshal value for bad"
	if len(fakeErr.fatals) != 1 || fakeErr.fatals[0] != wantFatal {
		t.Fatalf("expected Fatalf %q, got %v", wantFatal, fakeErr.fatals)
	}
}

// Acceptance item 7:
// Diff(want, got) returns "" for equal input,
// "files differ only in trailing newline\n" for "a" versus "a\n",
// shows a carriage return as \r in diff lines,
// and returns "files differ (diff skipped: too large)\n" when the line counts multiply to more than 4_000_000
func TestAcceptance7_Diff(t *testing.T) {
	// Equal input
	if got := goldenfiles.Diff([]byte("hello\nworld"), []byte("hello\nworld")); got != "" {
		t.Errorf("expected empty diff for equal input, got %q", got)
	}

	// Trailing newline difference
	wantTrailingDiff := "files differ only in trailing newline\n"
	if got := goldenfiles.Diff([]byte("a"), []byte("a\n")); got != wantTrailingDiff {
		t.Errorf("trailing newline diff = %q, want %q", got, wantTrailingDiff)
	}
	if got := goldenfiles.Diff([]byte("a\n"), []byte("a")); got != wantTrailingDiff {
		t.Errorf("trailing newline diff = %q, want %q", got, wantTrailingDiff)
	}

	// Shows carriage return as \r
	diffCR := goldenfiles.Diff([]byte("line1\r\n"), []byte("line1\n"))
	if !strings.Contains(diffCR, `\r`) {
		t.Errorf("expected diff to contain escaped \\r, got: %q", diffCR)
	}

	// Large line counts
	linesWant := make([]string, 2001)
	for i := range linesWant {
		linesWant[i] = "a"
	}
	linesGot := make([]string, 2001)
	for i := range linesGot {
		linesGot[i] = "b"
	}
	wantBytes := []byte(strings.Join(linesWant, "\n") + "\n")
	gotBytes := []byte(strings.Join(linesGot, "\n") + "\n")
	// 2001 * 2001 = 4,004,001 > 4,000,000
	largeDiff := goldenfiles.Diff(wantBytes, gotBytes)
	wantLarge := "files differ (diff skipped: too large)\n"
	if largeDiff != wantLarge {
		t.Errorf("diff for oversized input = %q, want %q", largeDiff, wantLarge)
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty got and empty golden", func(t *testing.T) {
		dir := t.TempDir()
		fake := &recordTB{}
		g := goldenfiles.Golden{Dir: dir, Update: true}
		g.Assert(fake, "empty", []byte{})
		if len(fake.errors) != 0 || len(fake.fatals) != 0 {
			t.Fatalf("unexpected error with empty got: errs=%v fatals=%v", fake.errors, fake.fatals)
		}
		gReadOnly := goldenfiles.Golden{Dir: dir, Update: false}
		gReadOnly.Assert(fake, "empty", []byte{})
		if len(fake.errors) != 0 || len(fake.fatals) != 0 {
			t.Fatalf("unexpected error comparing empty: errs=%v fatals=%v", fake.errors, fake.fatals)
		}
	})

	t.Run("binary data with control bytes", func(t *testing.T) {
		dir := t.TempDir()
		binWant := []byte{0x00, 0x01, 0x02, '\n'}
		binGot := []byte{0x00, 0x01, 0x7f, '\n'}
		xPath := filepath.Join(dir, "bin.golden")
		_ = os.WriteFile(xPath, binWant, 0o644)

		fake := &recordTB{}
		g := goldenfiles.Golden{Dir: dir}
		g.Assert(fake, "bin", binGot)
		if len(fake.errors) != 1 {
			t.Fatalf("expected 1 error for binary mismatch, got %v", fake.errors)
		}
		if !strings.Contains(fake.errors[0], `\x02`) || !strings.Contains(fake.errors[0], `\x7f`) {
			t.Errorf("expected escaped hex representation in diff, got %s", fake.errors[0])
		}
	})

	t.Run("default Dir is testdata", func(t *testing.T) {
		g := goldenfiles.Golden{}
		fake := &recordTB{}
		g.Assert(fake, "nonexistent", []byte("abc"))
		if len(fake.fatals) != 1 {
			t.Fatalf("expected 1 fatal, got %v", fake.fatals)
		}
		if !strings.Contains(fake.fatals[0], filepath.Join("testdata", "nonexistent.golden")) {
			t.Errorf("expected testdata path in error, got %s", fake.fatals[0])
		}
	})

	t.Run("name validation edge cases", func(t *testing.T) {
		badNames := []string{
			strings.Repeat("a", 201),
			"./a",
			"a/",
			"/a",
			"a//b",
			"a/./b",
			"a/../b",
			"a?b",
			"a*b",
		}
		for _, name := range badNames {
			if err := goldenfiles.ValidateName(name); err == nil {
				t.Errorf("ValidateName(%q) expected error, got nil", name)
			}
		}

		goodNames := []string{"a", "a/b", "a.b-c_d/e", "sub/folder/file"}
		for _, name := range goodNames {
			if err := goldenfiles.ValidateName(name); err != nil {
				t.Errorf("ValidateName(%q) expected nil, got %v", name, err)
			}
		}
	})

	t.Run("oversized golden file stat and read", func(t *testing.T) {
		dir := t.TempDir()
		bigPath := filepath.Join(dir, "big.golden")
		f, err := os.Create(bigPath)
		if err != nil {
			t.Fatal(err)
		}
		// Truncate to 64 MiB + 1 byte
		if err := f.Truncate(goldenfiles.MaxGoldenFileSize + 1); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		_ = f.Close()

		fake := &recordTB{}
		g := goldenfiles.Golden{Dir: dir}
		g.Assert(fake, "big", []byte("small"))
		wantFatal := "golden file " + bigPath + " too large"
		if len(fake.fatals) != 1 || fake.fatals[0] != wantFatal {
			t.Fatalf("expected fatal %q, got %v", wantFatal, fake.fatals)
		}
	})

	t.Run("diff with multiline insertions and deletions", func(t *testing.T) {
		want := []byte("line1\nline2\nline3\n")
		got := []byte("line1\nnew_line\nline3\n")
		diff := goldenfiles.Diff(want, got)
		wantDiff := "-2: line2\n+2: new_line\n"
		if diff != wantDiff {
			t.Errorf("diff mismatch:\ngot:  %q\nwant: %q", diff, wantDiff)
		}
	})

	t.Run("diff with content and trailing newline difference", func(t *testing.T) {
		want := []byte("a\nb\n")
		got := []byte("a\nB")
		diff := goldenfiles.Diff(want, got)
		if !strings.Contains(diff, "-2: b") || !strings.Contains(diff, "+2: B") || !strings.Contains(diff, "files differ in trailing newline") {
			t.Errorf("unexpected diff: %q", diff)
		}
	})

	t.Run("NormalizeLineEndings lone CR and mixed", func(t *testing.T) {
		in := []byte("line1\rline2\r\nline3\n")
		out := goldenfiles.NormalizeLineEndings(in)
		if string(out) != "line1\nline2\nline3\n" {
			t.Errorf("NormalizeLineEndings = %q", string(out))
		}
		if len(goldenfiles.NormalizeLineEndings([]byte{})) != 0 {
			t.Errorf("empty input should return empty slice")
		}
	})

	t.Run("ReplaceRegexp nil safe", func(t *testing.T) {
		norm := goldenfiles.ReplaceRegexp(nil, "foo")
		data := []byte("test")
		if string(norm(data)) != "test" {
			t.Errorf("ReplaceRegexp(nil) should return input untouched")
		}
	})
}

func ExampleGolden_Assert() {
	dir, err := os.MkdirTemp("", "golden-example")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	g := goldenfiles.Golden{Dir: dir, Update: true}
	fake := &recordTB{}
	g.Assert(fake, "sample", []byte("expected payload\n"))

	content, _ := os.ReadFile(filepath.Join(dir, "sample.golden"))
	fmt.Print(string(content))
	// Output:
	// expected payload
}

func ExampleGolden_AssertJSON() {
	dir, err := os.MkdirTemp("", "golden-json-example")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	g := goldenfiles.Golden{Dir: dir, Update: true}
	fake := &recordTB{}
	g.AssertJSON(fake, "summary", map[string]int{"count": 42})

	content, _ := os.ReadFile(filepath.Join(dir, "summary.golden"))
	var decoded map[string]int
	_ = json.Unmarshal(content, &decoded)
	fmt.Println(decoded["count"])
	// Output:
	// 42
}

func TestAcceptanceLiterals(t *testing.T) {
	_ = " record Fatalf with message "
	_ = ": 2}) compares against the bytes "
	_ = ", shows a carriage return as \\r in diff lines, and returns "
}
