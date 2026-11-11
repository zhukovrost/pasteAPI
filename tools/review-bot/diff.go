package main

import (
	"path"
	"sort"
	"strings"
)

// DiffFile — один файл из unified diff.
type DiffFile struct {
	Path     string
	IsBinary bool
	Added    int
	Removed  int
	Raw      string
}

// DiffStats — статистика по diff после фильтрации.
type DiffStats struct {
	Files   int
	Added   int
	Removed int
	Changed int
}

var ignoredPrefixes = []string{"dist/", "generated/", "vendor/", "node_modules/", "docs/"}

var ignoredNames = []string{
	"*.lock", "*.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	"*.min.js", "*.min.css",
}

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
	".ico": true, ".pdf": true, ".zip": true, ".gz": true, ".jar": true,
	".exe": true, ".so": true, ".dylib": true, ".a": true, ".woff": true,
	".woff2": true, ".mp4": true, ".mp3": true,
}

// ParseDiff разбирает unified diff на файлы и считает строки.
func ParseDiff(raw string) []DiffFile {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var (
		files  []DiffFile
		cur    DiffFile
		buf    strings.Builder
		has    bool
		inHunk bool
	)

	flush := func() {
		if !has {
			return
		}
		cur.Raw = buf.String()
		if strings.ContainsRune(cur.Raw, '\x00') {
			cur.IsBinary = true
		}
		if cur.Path == "" {
			cur.Path = pathFromHeader(cur.Raw)
		}
		files = append(files, cur)
		cur = DiffFile{}
		buf.Reset()
		has = false
		inHunk = false
	}

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			has = true
			buf.WriteString(line + "\n")
			continue
		}
		if !has {
			continue
		}
		buf.WriteString(line + "\n")

		switch {
		case strings.HasPrefix(line, "@@"):
			inHunk = true
		case !inHunk && strings.HasPrefix(line, "+++ "):
			if p := cleanPath(strings.TrimPrefix(line, "+++ ")); p != "" {
				cur.Path = p
			}
		case !inHunk && strings.HasPrefix(line, "--- "):
			if cur.Path == "" {
				if p := cleanPath(strings.TrimPrefix(line, "--- ")); p != "" {
					cur.Path = p
				}
			}
		case strings.HasPrefix(line, "Binary files "), strings.HasPrefix(line, "GIT binary patch"):
			cur.IsBinary = true
		case strings.HasPrefix(line, "+"):
			cur.Added++
		case strings.HasPrefix(line, "-"):
			cur.Removed++
		}
	}
	flush()

	return files
}

// FilterFiles убирает бинарники, lock-файлы и сгенерированный код.
func FilterFiles(files []DiffFile) (kept []DiffFile, ignored []string) {
	kept = make([]DiffFile, 0, len(files))
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		if f.IsBinary || isBinaryPath(f.Path) || isIgnoredPath(f.Path) {
			ignored = append(ignored, f.Path)
			continue
		}
		kept = append(kept, f)
	}
	sort.Strings(ignored)
	return kept, ignored
}

// ComputeStats считает статистику по оставленным файлам.
func ComputeStats(kept []DiffFile) DiffStats {
	s := DiffStats{Files: len(kept)}
	for _, f := range kept {
		s.Added += f.Added
		s.Removed += f.Removed
	}
	s.Changed = s.Added + s.Removed
	return s
}

// TooLarge сообщает, превышен ли порог автоматического ревью.
func TooLarge(s DiffStats, maxLines, maxFiles int) bool {
	return s.Changed > maxLines || s.Files > maxFiles
}

// TruncateFile вырезает середину слишком большого diff одного файла.
func TruncateFile(f DiffFile, maxBytes int) DiffFile {
	if maxBytes <= 0 || len(f.Raw) <= maxBytes {
		return f
	}
	f.Raw = truncateMiddle(f.Raw, maxBytes)
	return f
}

func truncateMiddle(s string, max int) string {
	const marker = "\n… diff truncated …\n"
	headLimit := max * 3 / 5
	tailLimit := max - headLimit - len(marker)
	if tailLimit < 0 {
		tailLimit = 0
	}

	head := s[:headLimit]
	if i := strings.LastIndexByte(head, '\n'); i >= 0 {
		head = head[:i+1]
	}

	tail := ""
	if tailLimit > 0 {
		tail = s[len(s)-tailLimit:]
		if i := strings.IndexByte(tail, '\n'); i >= 0 {
			tail = tail[i+1:]
		}
	}
	return head + marker + tail
}

func isIgnoredPath(p string) bool {
	for _, prefix := range ignoredPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	base := path.Base(p)
	for _, glob := range ignoredNames {
		if ok, _ := path.Match(glob, base); ok {
			return true
		}
	}
	return false
}

func isBinaryPath(p string) bool {
	return binaryExts[strings.ToLower(path.Ext(p))]
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(p, `"`) {
		p = strings.Trim(p, `"`)
		p = strings.ReplaceAll(p, `\"`, `"`)
	}
	if i := strings.IndexByte(p, '\t'); i >= 0 {
		p = p[:i]
	}
	switch {
	case strings.HasPrefix(p, "b/"):
		p = p[2:]
	case strings.HasPrefix(p, "a/"):
		p = p[2:]
	}
	return p
}

func pathFromHeader(raw string) string {
	first := raw
	if i := strings.IndexByte(raw, '\n'); i >= 0 {
		first = raw[:i]
	}
	if i := strings.LastIndex(first, " b/"); i >= 0 {
		return cleanPath(first[i+1:])
	}
	return ""
}
