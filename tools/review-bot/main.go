package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	maxLines       = 1500
	maxFiles       = 40
	maxComments    = 20
	maxFileBytes   = 30000
	maxPromptBytes = 120000
	requestTimeout = 240 * time.Second
)

// Comment — одно замечание модели.
type Comment struct {
	File     string `json:"file"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Reason   string `json:"reason"`
}

type ReviewResult struct {
	Comments []Comment `json:"comments"`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("ai-review: ")

	diffPath := flag.String("diff", "", "unified diff file")
	instructionsPath := flag.String("instructions", "", "review instructions markdown")
	outPath := flag.String("out", "", "markdown result file")
	flag.Parse()

	if *diffPath == "" || *instructionsPath == "" || *outPath == "" {
		log.Fatal("-diff, -instructions and -out are required")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	model := envOr("GEMINI_MODEL", "gemini-3.8-flash")

	log.Printf("start: model=%s, timeout=%s", model, requestTimeout)

	markdown, err := Execute(*diffPath, *instructionsPath, model, nil, apiKey)
	if err != nil {
		log.Print(err)
		markdown = renderFailure(err.Error(), apiKey)
	}

	if err := os.WriteFile(*outPath, []byte(markdown), 0o644); err != nil {
		log.Printf("write %s: %v", *outPath, err)
		os.Exit(1)
	}
}

// Execute — вся логика бота. Ошибки модели превращаются в fallback-комментарий.
func Execute(diffPath, instructionsPath, model string, reviewer Reviewer, apiKey string) (string, error) {
	rawDiff, err := os.ReadFile(diffPath)
	if err != nil {
		return "", fmt.Errorf("read diff: %w", err)
	}

	kept, ignored := FilterFiles(ParseDiff(string(rawDiff)))
	stats := ComputeStats(kept)
	log.Printf("diff: %d files, +%d/-%d lines, ignored %d", stats.Files, stats.Added, stats.Removed, len(ignored))

	if stats.Files == 0 {
		log.Print("stop: no reviewable changes after filtering")
		return "Нет текстовых изменений для ревью после исключений.", nil
	}
	if TooLarge(stats, maxLines, maxFiles) {
		log.Printf("stop: diff too large (%d lines, %d files)", stats.Changed, stats.Files)
		return renderSkipped(stats), nil
	}

	checklist, err := os.ReadFile(instructionsPath)
	if err != nil {
		return "", fmt.Errorf("read instructions: %w", err)
	}
	if strings.TrimSpace(string(checklist)) == "" {
		return "", fmt.Errorf("review instructions are empty")
	}
	log.Printf("instructions: %d bytes", len(checklist))

	files := make([]DiffFile, 0, len(kept))
	for _, f := range kept {
		files = append(files, TruncateFile(f, maxFileBytes))
	}

	systemPrompt := BuildSystemPrompt(string(checklist))
	userPrompt := BuildUserPrompt(files, stats, maxPromptBytes)
	log.Printf("prompt: system=%d bytes, user=%d bytes", len(systemPrompt), len(userPrompt))

	if reviewer == nil {
		if strings.TrimSpace(apiKey) == "" {
			log.Print("stop: GEMINI_API_KEY is empty")
			return renderFailure("не настроен GEMINI_API_KEY", apiKey), nil
		}
		r, err := NewGeminiReviewer(apiKey, model)
		if err != nil {
			log.Printf("create gemini client: %v", err)
			return renderFailure("не удалось создать клиента модели: "+err.Error(), apiKey), nil
		}
		reviewer = r
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	start := time.Now()
	rawOut, err := reviewer.Review(ctx, systemPrompt, userPrompt)
	log.Printf("model call finished in %s", time.Since(start).Round(time.Millisecond))
	if err != nil {
		log.Printf("stop: model failed: %v", err)
		return renderFailure("модель недоступна: "+err.Error(), apiKey), nil
	}
	log.Printf("model response: %d bytes", len(rawOut))

	result, err := parseReview(rawOut)
	if err != nil {
		log.Printf("stop: invalid model output: %v", err)
		return renderFailure("модель вернула невалидный JSON", apiKey), nil
	}
	log.Printf("parsed comments: %d", len(result.Comments))

	result = normalizeResult(result, kept)
	log.Printf("comments after normalization: %d", len(result.Comments))
	return renderReview(result, stats), nil
}

func parseReview(raw string) (ReviewResult, error) {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}

	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return ReviewResult{}, fmt.Errorf("no JSON object found")
	}

	var r ReviewResult
	if err := json.Unmarshal([]byte(s[start:end+1]), &r); err != nil {
		return ReviewResult{}, err
	}
	if r.Comments == nil {
		return ReviewResult{}, fmt.Errorf("missing comments array")
	}
	return r, nil
}

func normalizeResult(r ReviewResult, files []DiffFile) ReviewResult {
	allowed := make(map[string]bool, len(files))
	for _, f := range files {
		allowed[f.Path] = true
	}

	out := ReviewResult{Comments: make([]Comment, 0, len(r.Comments))}
	seen := make(map[string]bool)

	for _, c := range r.Comments {
		c.File = normalizePath(c.File, allowed)
		c.Severity = normalizeSeverity(c.Severity)
		c.Message = strings.TrimSpace(c.Message)
		c.Reason = strings.TrimSpace(c.Reason)

		if c.File == "" || c.Message == "" || c.Severity == "" || !allowed[c.File] {
			continue
		}
		key := c.File + "|" + c.Severity + "|" + c.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Comments = append(out.Comments, c)
	}

	if len(out.Comments) > maxComments {
		out.Comments = out.Comments[:maxComments]
	}
	return out
}

func normalizePath(p string, allowed map[string]bool) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"`)
	if allowed[p] {
		return p
	}
	for _, prefix := range []string{"b/", "a/"} {
		if strings.HasPrefix(p, prefix) && allowed[p[2:]] {
			return p[2:]
		}
	}
	return cleanPath(p)
}

func normalizeSeverity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "error", "critical", "fatal":
		return "error"
	case "warning", "warn":
		return "warning"
	case "info", "suggestion", "note", "style":
		return "info"
	default:
		return ""
	}
}

func renderReview(r ReviewResult, stats DiffStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## AI review\n\nФайлов: %d, строк: +%d/-%d\n", stats.Files, stats.Added, stats.Removed)

	if len(r.Comments) == 0 {
		b.WriteString("\nЗамечаний не найдено.\n")
		return b.String()
	}

	for _, c := range r.Comments {
		fmt.Fprintf(&b, "\n### [%s] %s\n\n**Проблема:** %s\n\n**Почему:** %s\n",
			c.Severity, c.File, c.Message, c.Reason)
	}
	return b.String()
}

func renderSkipped(stats DiffStats) string {
	return fmt.Sprintf(
		"PR слишком большой для автоматического ревью: %d строк, %d файлов (порог %d/%d). Review пропущен.",
		stats.Changed, stats.Files, maxLines, maxFiles)
}

func renderFailure(reason, apiKey string) string {
	return "AI review не выполнен: " + sanitize(reason, apiKey)
}

func sanitize(s, secret string) string {
	if secret != "" {
		s = strings.ReplaceAll(s, secret, "***")
	}
	runes := []rune(s)
	if len(runes) > 500 {
		s = string(runes[:500]) + "…"
	}
	return s
}

func envOr(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
