package main

import (
	"strings"
	"testing"
)

const testDiff = `diff --git a/internal/http/v1/pastes.go b/internal/http/v1/pastes.go
--- a/internal/http/v1/pastes.go
+++ b/internal/http/v1/pastes.go
@@ -10,2 +10,3 @@ func handler() {
-old line
+new line
+added line
 context
diff --git a/go.sum b/go.sum
--- a/go.sum
+++ b/go.sum
@@ -1 +1 @@
-old
+new
diff --git a/logo.png b/logo.png
new file mode 100644
Binary files /dev/null and b/logo.png differ
diff --git a/dist/app.js b/dist/app.js
--- a/dist/app.js
+++ b/dist/app.js
@@ -1 +1 @@
-old
+new
`

func TestParseAndFilterDiff(t *testing.T) {
	files := ParseDiff(testDiff)
	if len(files) != 4 {
		t.Fatalf("expected 4 parsed files, got %d", len(files))
	}

	kept, ignored := FilterFiles(files)
	if len(kept) != 1 || kept[0].Path != "internal/http/v1/pastes.go" {
		t.Fatalf("unexpected kept files: %+v", kept)
	}
	if strings.Join(ignored, ",") != "dist/app.js,go.sum,logo.png" {
		t.Fatalf("unexpected ignored files: %v", ignored)
	}

	stats := ComputeStats(kept)
	if stats.Files != 1 || stats.Added != 2 || stats.Removed != 1 || stats.Changed != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestHunkContentStartingWithPluses(t *testing.T) {
	files := ParseDiff(`diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1,2 @@
-old
+++ not a header
`)
	if len(files) != 1 || files[0].Path != "a.go" {
		t.Fatalf("hunk content must not override path: %+v", files)
	}
	if files[0].Added != 1 || files[0].Removed != 1 {
		t.Fatalf("unexpected stats: +%d/-%d", files[0].Added, files[0].Removed)
	}
}

func TestCleanPathKeepsInnerPrefix(t *testing.T) {
	if got := cleanPath("b/a/foo.go"); got != "a/foo.go" {
		t.Fatalf("got %q", got)
	}
}

func TestTooLargeBoundaries(t *testing.T) {
	cases := []struct {
		name string
		s    DiffStats
		want bool
	}{
		{"exact lines", DiffStats{Changed: 1500, Files: 1}, false},
		{"over lines", DiffStats{Changed: 1501, Files: 1}, true},
		{"exact files", DiffStats{Changed: 1, Files: 40}, false},
		{"over files", DiffStats{Changed: 1, Files: 41}, true},
	}
	for _, tc := range cases {
		if got := TooLarge(tc.s, 1500, 40); got != tc.want {
			t.Errorf("%s: TooLarge = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIgnoredPaths(t *testing.T) {
	for _, p := range []string{"go.sum", "package-lock.json", "dist/app.js", "generated/api.go", "docs/docs.go", "app.min.js"} {
		if !isIgnoredPath(p) {
			t.Errorf("%q should be ignored", p)
		}
	}
	if isIgnoredPath("internal/http/v1/pastes.go") {
		t.Errorf("normal file must not be ignored")
	}
}

func TestTruncateFile(t *testing.T) {
	f := DiffFile{Path: "big.go", Raw: strings.Repeat("+line\n", 1000)}
	got := TruncateFile(f, 500)
	if len(got.Raw) > 600 || !strings.Contains(got.Raw, "diff truncated") {
		t.Fatalf("unexpected truncation result: %d bytes", len(got.Raw))
	}
}
