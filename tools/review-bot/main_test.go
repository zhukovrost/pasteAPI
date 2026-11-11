package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeReviewer struct {
	out        string
	err        error
	calls      int
	userPrompt string
}

func (f *fakeReviewer) Review(_ context.Context, _, userPrompt string) (string, error) {
	f.calls++
	f.userPrompt = userPrompt
	return f.out, f.err
}

func setup(t *testing.T, diff string) (diffPath, instructionsPath string) {
	t.Helper()
	dir := t.TempDir()
	diffPath = filepath.Join(dir, "pr.diff")
	instructionsPath = filepath.Join(dir, "instructions.md")
	if err := os.WriteFile(diffPath, []byte(diff), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instructionsPath, []byte("SQL injection -> error"), 0o644); err != nil {
		t.Fatal(err)
	}
	return diffPath, instructionsPath
}

func makeDiff(files, addedLines int) string {
	var b strings.Builder
	for i := 0; i < files; i++ {
		fmt.Fprintf(&b, "diff --git a/file%03d.go b/file%03d.go\n--- a/file%03d.go\n+++ b/file%03d.go\n@@ -1 +1 @@\n", i, i, i, i)
		for j := 0; j < addedLines; j++ {
			b.WriteString("+line\n")
		}
	}
	return b.String()
}

func TestSkipsLargePR(t *testing.T) {
	diffPath, instr := setup(t, makeDiff(41, 0))
	fake := &fakeReviewer{out: `{"comments":[]}`}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 {
		t.Fatalf("model must not be called, calls=%d", fake.calls)
	}
	if !strings.Contains(md, "пропущен") {
		t.Fatalf("unexpected result: %s", md)
	}
}

func TestNoReviewableChanges(t *testing.T) {
	diffPath, instr := setup(t, "diff --git a/go.sum b/go.sum\n--- a/go.sum\n+++ b/go.sum\n@@ -1 +1 @@\n-a\n+b\n")
	fake := &fakeReviewer{out: `{"comments":[]}`}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 0 || !strings.Contains(md, "Нет текстовых изменений") {
		t.Fatalf("calls=%d, result=%s", fake.calls, md)
	}
}

func TestRendersComments(t *testing.T) {
	diffPath, instr := setup(t, makeDiff(1, 2))
	fake := &fakeReviewer{out: "```json\n" +
		`{"comments":[{"file":"file000.go","severity":"critical","message":"bad SQL","reason":"#1"}]}` +
		"\n```"}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[error] file000.go", "bad SQL", "#1"} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in output:\n%s", want, md)
		}
	}
}

func TestInvalidJSONFallback(t *testing.T) {
	diffPath, instr := setup(t, makeDiff(1, 2))
	fake := &fakeReviewer{out: "not a json"}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "невалидный JSON") {
		t.Fatalf("unexpected result: %s", md)
	}
}

func TestModelErrorFallback(t *testing.T) {
	fastRetries(t)
	diffPath, instr := setup(t, makeDiff(1, 2))
	fake := &fakeReviewer{err: errors.New("timeout")}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "модель недоступна") {
		t.Fatalf("unexpected result: %s", md)
	}
	if fake.calls != modelAttempts {
		t.Fatalf("expected %d attempts, got %d", modelAttempts, fake.calls)
	}
}

type flakyReviewer struct {
	calls int
}

func (f *flakyReviewer) Review(_ context.Context, _, _ string) (string, error) {
	f.calls++
	if f.calls < modelAttempts {
		return "", errors.New("504 deadline exceeded")
	}
	return `{"comments":[]}`, nil
}

func TestRetryRecoversAfterTransientError(t *testing.T) {
	fastRetries(t)
	diffPath, instr := setup(t, makeDiff(1, 2))
	fake := &flakyReviewer{}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != modelAttempts {
		t.Fatalf("expected %d attempts, got %d", modelAttempts, fake.calls)
	}
	if !strings.Contains(md, "Замечаний не найдено") {
		t.Fatalf("unexpected result: %s", md)
	}
}

func fastRetries(t *testing.T) {
	t.Helper()
	orig := retryBackoff
	retryBackoff = []time.Duration{0, 0}
	t.Cleanup(func() { retryBackoff = orig })
}

func TestMissingAPIKeyFallback(t *testing.T) {
	diffPath, instr := setup(t, makeDiff(1, 2))

	md, err := Execute(diffPath, instr, "test", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "GEMINI_API_KEY") {
		t.Fatalf("unexpected result: %s", md)
	}
}

func TestDropsUnknownFiles(t *testing.T) {
	diffPath, instr := setup(t, makeDiff(1, 2))
	fake := &fakeReviewer{out: `{"comments":[{"file":"other.go","severity":"error","message":"made up","reason":"x"}]}`}

	md, err := Execute(diffPath, instr, "test", fake, "key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Замечаний не найдено") {
		t.Fatalf("unexpected result: %s", md)
	}
}

func TestPromptFiltersIgnoredFiles(t *testing.T) {
	diff := makeDiff(1, 1) + "diff --git a/go.sum b/go.sum\n--- a/go.sum\n+++ b/go.sum\n@@ -1 +1 @@\n-a\n+b\n"
	diffPath, instr := setup(t, diff)
	fake := &fakeReviewer{out: `{"comments":[]}`}

	if _, err := Execute(diffPath, instr, "test", fake, "key"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fake.userPrompt, "diff --git a/file000.go") {
		t.Fatalf("changed file missing from prompt")
	}
	if strings.Contains(fake.userPrompt, "go.sum") {
		t.Fatalf("ignored file leaked into prompt")
	}
}

func TestSanitizeRemovesKey(t *testing.T) {
	secret := "AIzaSECRETKEY1234567890"
	got := sanitize("error: "+secret, secret)
	if strings.Contains(got, secret) {
		t.Fatalf("secret leaked: %s", got)
	}
}
